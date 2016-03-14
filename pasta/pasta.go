package main

import "fmt"
import "os"
import "io"
import "runtime"
import "runtime/pprof"

import "strconv"
import "strings"
import "time"
import "bufio"

import "github.com/abeconnelly/autoio"
import "github.com/codegangsta/cli"

import "github.com/abeconnelly/pasta"

var VERSION_STR string = "0.1.0"
var gVerboseFlag bool

var gProfileFlag bool
var gProfileFile string = "pasta.pprof"

var gMemProfileFlag bool
var gMemProfileFile string = "pasta.mprof"

var gFullRefSeqFlag bool = true
var gFullNocSeqFlag bool = true

var g_debug bool = false

func echo_stream(stream *bufio.Reader) {
  var e error
  var ch byte
  out := bufio.NewWriter(os.Stdout)
  for ch,e = stream.ReadByte() ; e==nil ; ch,e = stream.ReadByte() {
    out.WriteByte(ch)
  }
}

// to lower [a-z]
//
func _tolch(A byte) byte {
  z := A
  if A >= 'A' && A <= 'Z' {
    z = A - 'A' + 'a'
  } else {
    z = A
  }
  return z
}


type VarDiff struct {
  Type      string
  RefStart  int
  RefLen    int
  RefSeq    string
  AltSeq    []string
}


func InterleaveStreamToVarDiff(stream *bufio.Reader, N ...int) ([]VarDiff, error) {
  n:=-1
  if len(N)>0 { n=N[0] }
  if n<=0 { n=-1 }

  vardiff := make([]VarDiff, 0, 16)

  alt0 := []byte{}
  alt1 := []byte{}
  refseq := []byte{}

  ref_start := 0
  ref0_len := 0
  ref1_len := 0

  is_refn_cur := true
  is_refn_prv := true

  is_first_pass := true

  stream0_pos:=0
  stream1_pos:=0

  for (n<0) || (n>0) {

    is_ref0 := false
    is_ref1 := false

    ch0,e0 := stream.ReadByte()
    ch1,e1 := stream.ReadByte()

    stream0_pos++
    stream1_pos++

    if e0!=nil && e1!=nil { break }

    // special case: nop
    //
    if ch0=='.' && ch1=='.' { continue }

    dbp0 := pasta.RefDelBP[ch0]
    dbp1 := pasta.RefDelBP[ch1]

    if ch0=='a' || ch0=='c' || ch0=='g' || ch0=='t' || ch0=='n' || ch0=='N' { is_ref0=true }
    if ch1=='a' || ch1=='c' || ch1=='g' || ch1=='t' || ch1=='n' || ch1=='N' { is_ref1=true }

    if is_ref0 && is_ref1 {
      is_refn_cur = true
    } else {
      is_refn_cur = false
    }

    if is_first_pass {
      is_refn_prv = is_refn_cur
      is_first_pass = false

      if !is_ref0 || !is_ref1 {
        if bp,ok := pasta.RefMap[ch0] ; ok {
          refseq = append(refseq, bp)
        } else if bp, ok := pasta.RefMap[ch1] ; ok {
          refseq = append(refseq, bp)
        }
      } else if gFullRefSeqFlag {
        if bp,ok := pasta.RefMap[ch0] ; ok {
          refseq = append(refseq, bp)
        } else if bp, ok := pasta.RefMap[ch1] ; ok {
          refseq = append(refseq, bp)
        }
      }

      ref0_len+=dbp0
      ref1_len+=dbp1

      continue
    }

    // assert ch0==ch1 if they're both reference
    //
    if is_ref0 && is_ref1 && ch0!=ch1 {
      return nil, fmt.Errorf(fmt.Sprintf("ERROR: stream position (%d,%d), stream0 token %c (%d), stream1 token %c (%d)",
        stream0_pos, stream1_pos, ch0, ch0, ch1, ch1))
    }

    if !is_refn_cur && is_refn_prv {

      if gFullRefSeqFlag {
        vardiff = append(vardiff, VarDiff{"REF", ref_start, ref0_len, string(refseq), []string{"",""}})
      } else {
        vardiff = append(vardiff, VarDiff{"REF", ref_start, ref0_len, "", []string{"",""}})
      }
      if n>0 { n-- }

      ref_start += ref0_len

      ref0_len=0
      ref1_len=0

      alt0 = alt0[0:0]
      alt1 = alt1[0:0]
      refseq = refseq[0:0]

    } else if is_refn_cur && !is_refn_prv {

      vardiff = append(vardiff, VarDiff{"ALT", ref_start, ref0_len, string(refseq), []string{string(alt0), string(alt1)}})
      if n>0 { n-- }

      ref_start += ref0_len

      ref0_len=0
      ref1_len=0

      alt0 = alt0[0:0]
      alt1 = alt1[0:0]
      refseq = refseq[0:0]
    } else {
      // The current state matches the previous state.
      // Either both the current tokens are non-ref as well as the previous tokens
      // or both the current token and previous tokens are ref.
    }

    if !is_ref0 || !is_ref1 {
      if bp,ok := pasta.RefMap[ch0] ; ok {
        refseq = append(refseq, bp)
      } else if bp, ok := pasta.RefMap[ch1] ; ok {
        refseq = append(refseq, bp)
      }

      if bp_val,ok := pasta.AltMap[ch0] ; ok { alt0 = append(alt0, bp_val) }
      if bp_val,ok := pasta.AltMap[ch1] ; ok { alt1 = append(alt1, bp_val) }

    } else if gFullRefSeqFlag {
      if bp,ok := pasta.RefMap[ch0] ; ok {
        refseq = append(refseq, bp)
      } else if bp, ok := pasta.RefMap[ch1] ; ok {
        refseq = append(refseq, bp)
      }

      if bp_val,ok := pasta.AltMap[ch0] ; ok { alt0 = append(alt0, bp_val) }
      if bp_val,ok := pasta.AltMap[ch1] ; ok { alt1 = append(alt1, bp_val) }

    }

    ref0_len+=dbp0
    ref1_len+=dbp1

    is_refn_prv = is_refn_cur

  }

  // Final diff line
  //
  if is_refn_prv {
    if gFullRefSeqFlag {
      vardiff = append(vardiff, VarDiff{"REF", ref_start, ref0_len, string(refseq), []string{"",""}})
    } else {
      vardiff = append(vardiff, VarDiff{"REF", ref_start, ref0_len, string(""), []string{"",""}})
    }
  } else if !is_refn_prv {
    vardiff = append(vardiff, VarDiff{"ALT", ref_start, ref0_len, string(refseq), []string{string(alt0), string(alt1)}})
  }

  return vardiff, nil
}

/*
type ControlMessage struct {
  Type    int
  N       int
  NBytes  int

  Chrom   string
  RefPos  int
  RefLen  int

  Comment string
}
*/

/*
func control_message_print(msg *pasta.ControlMessage, out *bufio.Writer) {

  if msg.Type == REF {
    out.WriteString(fmt.Sprintf(">R{%d}", msg.N))
  } else if msg.Type == POS {
    out.WriteString(fmt.Sprintf(">P{%d}", msg.RefPos))
  } else if msg.Type == NOC {
    out.WriteString(fmt.Sprintf(">P{%d}", msg.N))
  } else if msg.Type == CHROM {
    out.WriteString(fmt.Sprintf(">C{%s}", msg.Chrom))
  } else if msg.Type == COMMENT {
    out.WriteString(fmt.Sprintf(">#{%s}", msg.Comment))
  }

}
*/

/*
func process_control_message(stream *bufio.Reader) (pasta.ControlMessage, error) {
  var msg pasta.ControlMessage

  ch,e := stream.ReadByte()
  msg.NBytes++

  if e!=nil { return msg, e }

  if ch=='R' {
    msg.Type = REF
  } else if ch == 'N' {
    msg.Type = NOC
  } else if ch == 'C' {
    msg.Type = CHROM
  } else if ch == 'P' {
    msg.Type = POS
  } else if ch == '#' {
    msg.Type = COMMENT
  } else {
    return msg, fmt.Errorf("Invalid control character %c", ch)
  }

  ch,e = stream.ReadByte()
  msg.NBytes++
  if e!=nil { return msg, e }
  if ch!='{' { return msg, fmt.Errorf("Invalid control block start (expected '{' got %c)", ch) }

  field_str := make([]byte, 0, 32)

  for (e==nil) && (ch!='}') {
    ch,e = stream.ReadByte()
    msg.NBytes++
    if e!=nil { return msg, e }
    field_str = append(field_str, ch)
  }

  n:=len(field_str)

  if (n==0) || (n==1) {
    msg.N = 0
    return msg, nil
  }

  field_str = field_str[:n-1]

  if msg.Type == REF || msg.Type == NOC || msg.Type == POS {
    _i,err := strconv.Atoi(string(field_str))
    if err!=nil { return msg, err }

    if msg.Type == POS {
      msg.RefPos = int(_i)
    } else {
      msg.N = int(_i)
    }
  } else if msg.Type == CHROM {
    msg.Chrom = string(field_str)
  } else if msg.Type == COMMENT {
    msg.Comment = string(field_str)
  }
  return msg, nil

}
*/

/*
const(
  BEG = iota  // 0
  REF = iota
  NOC = iota
  ALT = iota
  MSG = iota
  MSG_REF_NOC = iota
  MSG_CHROM = iota
  MSG_POS = iota
  FIN = iota
  SNP = iota
  INDEL = iota
  NOREF = iota

  CHROM = iota
  POS = iota
  COMMENT = iota
)
*/


type RefVarInfo struct {
  Type int
  MessageType int
  RefSeqFlag bool
  NocSeqFlag bool
  Out io.Writer
  Msg pasta.ControlMessage
  RefBP byte

  Chrom string
}

type GVCFVarInfo struct {
  Type int
  MessageType int
  RefSeqFlag bool
  NocSeqFlag bool
  Out io.Writer
  Msg pasta.ControlMessage
  RefBP byte

  PrintHeader bool
  Header string
  Reference string
}

type RefVarProcesser func(int,int,int,[]byte,[][]byte,interface{}) error

func gvcf_header(info *GVCFVarInfo) string {
  reference_string := info.Reference
  t := time.Now()
  hdr := `##fileDate=` + t.Format(time.RFC3339) + "\n" +
`##source=pasta-to-gvcf
##description="Converted from a PASTA stream to gVCF"
##reference=` + reference_string + "\n" +
`##FILTER=<ID=NOCALL,Description="Some or all of this record had no sequence call">
##FILTER=<ID=VQLOW,Description="Some or all of this sequence call marked as low variant quality">
##FILTER=<ID=AMBIGUOUS,Description="Some or all of this sequence call marked as ambiguous">
##FORMAT=<ID=GT,Number=1,Type=String,Description="Genotype">
##INFO=<ID=END,Number=1,Type=Integer,Description="Stop position of the interval">`

  vcf_col := []string{ "CHROM", "POS", "ID", "REF", "ALT", "QUAL", "FILTER", "INFO", "FORMAT", "SAMPLE" }
  hdr = hdr + "\n#" + strings.Join(vcf_col, "\t")

  return hdr
}

func gvcf_refvar_printer(vartype int, ref_start, ref_len int, refseq []byte, altseq [][]byte, info_if interface{}) error {

  info := info_if.(*GVCFVarInfo) ; _ = info

  if info.PrintHeader {
    fmt.Printf("%s\n", gvcf_header(info))
    info.PrintHeader = false
  }

  chrom_field := "Unk"
  id_field    := "."

  r_field     := "x" ; _ = r_field
  alt_field   := "." ; _ = alt_field

  qual_field  := "."
  filt_field  := "PASS"
  info_field  := "."
  fmt_field   := "GT"
  samp_field  := "0/0"


  ref_bp := info.RefBP

  out := os.Stdout

  if vartype == pasta.REF {

    info_field = fmt.Sprintf("END=%d", ref_start+ref_len+1)
    out.Write( []byte(fmt.Sprintf("%s\t%d\t%s\t%c\t%s\t%s\t%s\t%s\t%s\t%s\n",
      chrom_field,
      ref_start+1, id_field,
      ref_bp, alt_field,
      qual_field, filt_field,
      info_field, fmt_field, samp_field)) )

  } else if vartype == pasta.NOC {
    filt_field = "NOCALL"
    samp_field = "./."

    info_field = fmt.Sprintf("END=%d", ref_start+ref_len+1)
    out.Write( []byte(fmt.Sprintf("%s\t%d\t%s\t%c\t%s\t%s\t%s\t%s\t%s\t%s\n",
      chrom_field,
      ref_start+1, id_field,
      ref_bp, alt_field,
      qual_field, filt_field,
      info_field, fmt_field, samp_field)) )

  } else if vartype == pasta.ALT {

    snp_flag := true
    if len(refseq)==1 {
      for i:=0; i<len(altseq); i++ {
        if len(altseq[i])!=1 {
          snp_flag = false
          break
        }
      }
      if snp_flag { ref_bp = refseq[0] }
    } else {
      snp_flag = false
    }

    out.Write( []byte(fmt.Sprintf("%s\t%d\t%s\t%c\t", chrom_field, ref_start+1, id_field, ref_bp)) )
    for i:=0; i<len(altseq); i++ {
      if i>0 { out.Write([]byte(",")) }
      out.Write( []byte(altseq[i]) )
    }
    out.Write( []byte(fmt.Sprintf("\t%s\t%s\t%s\t%s\t%s\n", qual_field, filt_field, info_field, fmt_field, samp_field)) )

  } else if vartype == pasta.MSG {

    /*
    if info.Msg.Type == REF {
      out.Write( []byte(fmt.Sprintf("ref\t%d\t%d\t.(msg)\n", ref_start, ref_start+info.Msg.N)) )
    } else if info.Msg.Type == NOC {
      out.Write( []byte(fmt.Sprintf("noc\t%d\t%d\t.(msg)\n", ref_start, ref_start+info.Msg.N)) )
    }
    */

  }

  return nil

}

type VarLine struct {
  Type    int
  Chrom   string
  RefPos  int
  RefLen  int
  RefSeq  string
  AltSeq  []string
  GT      []string
}

var g_vcf_buffer []VarLine

func simple_vcf_printer(vartype int, ref_start, ref_len int, refseq []byte, altseq [][]byte, info_if interface{}) error {

  info := info_if.(*RefVarInfo)

  out := os.Stdout

  //DEBUG
  fmt.Printf(">>>> vartype %d, ref_start %d, ref_len %d, refseq %s, altseq %v\n",
    vartype, ref_start, ref_len, refseq, altseq)

  if vartype == pasta.REF {


    g_vcf_buffer = append(g_vcf_buffer,
      VarLine{Type:pasta.REF,
              Chrom: info.Msg.Chrom,
              RefPos:ref_start,
              RefLen:ref_len,
              RefSeq:string(refseq),
              AltSeq:nil,
              GT:[]string{"0/0"}})

  } else if vartype == pasta.NOC {

    s_altseq := []string{}
    for i:=0; i<len(altseq); i++ {
      s_altseq = append(s_altseq, string(altseq[i]))
    }

    g_vcf_buffer = append(g_vcf_buffer,
      VarLine{Type: pasta.NOC,
              Chrom: info.Msg.Chrom,
              RefPos:ref_start,
              RefLen:ref_len,
              RefSeq:string(refseq),
              AltSeq:nil,
              GT:[]string{"./."}})

  } else if vartype == pasta.ALT {

    s_altseq := []string{}
    for i:=0; i<len(altseq); i++ {
      s_altseq = append(s_altseq, string(altseq[i]))
    }

    gt_string := fmt.Sprintf("%d/%d", -1,-2)

    g_vcf_buffer = append(g_vcf_buffer,
      VarLine{Type: pasta.ALT,
              Chrom: info.Msg.Chrom,
              RefPos:ref_start,
              RefLen:ref_len,
              RefSeq:string(refseq),
              AltSeq:s_altseq,
              GT:[]string{gt_string}})

  } else if vartype == pasta.MSG {

    if info.Msg.Type == pasta.REF {

      g_vcf_buffer = append(g_vcf_buffer,
        VarLine{Type: pasta.REF,
                Chrom: info.Msg.Chrom,
                RefPos:ref_start,
                RefLen:info.Msg.N,
                RefSeq:string(refseq),
                AltSeq:nil,
                GT:[]string{"."}})

      out.Write( []byte(fmt.Sprintf("ref\t%d\t%d\t.(msg)\n", ref_start, ref_start+info.Msg.N)) )
    } else if info.Msg.Type == pasta.NOC {

      g_vcf_buffer = append(g_vcf_buffer,
        VarLine{Type: pasta.NOC,
                Chrom: info.Msg.Chrom,
                RefPos:ref_start,
                RefLen:info.Msg.N,
                RefSeq:string(refseq),
                AltSeq:nil,
                GT:[]string{"."}})
    }

  }

  if len(g_vcf_buffer) > 2 {

    fmt.Printf("??\n")

    //chrom_field := "Unk"
    id_field    := "."

    r_field     := "x" ; _ = r_field
    alt_field   := "." ; _ = alt_field

    qual_field  := "."
    filt_field  := "PASS"
    info_field  := "."
    fmt_field   := "GT"
    samp_field  := "0/0"


    if (g_vcf_buffer[0].Type == pasta.REF) && (g_vcf_buffer[1].Type == pasta.ALT) {

      //DEBUG
      fmt.Printf("cpA\n")


      min_len,max_len := len(g_vcf_buffer[1].RefSeq), len(g_vcf_buffer[1].RefSeq)
      for i:=0; i<len(g_vcf_buffer[1].AltSeq); i++ {
        if i==0 {
          min_len,max_len = len(g_vcf_buffer[1].AltSeq[0]), len(g_vcf_buffer[1].AltSeq[0])
          continue
        }
        if min_len > len(g_vcf_buffer[1].AltSeq[i]) { min_len = len(g_vcf_buffer[1].AltSeq[i]) }
        if max_len < len(g_vcf_buffer[1].AltSeq[i]) { max_len = len(g_vcf_buffer[1].AltSeq[i]) }
      }

      if (min_len==1) && (max_len==1) {

        // REF then SNP

        t:=g_vcf_buffer[0]

        info_field = fmt.Sprintf("END=%d", t.RefPos+t.RefLen+1)
        out.Write( []byte(fmt.Sprintf("%s\t%d\t%s\t%c\t%s\t%s\t%s\t%s\t%s\t%s\n",
          t.Chrom,
          t.RefPos+1, id_field,
          t.RefSeq[0], alt_field,
          qual_field, filt_field,
          info_field, fmt_field, samp_field)) )

        t = g_vcf_buffer[1]

        out.Write( []byte(fmt.Sprintf("%s\t%d\t%s\t%c\t", t.Chrom, t.RefPos+1, id_field, t.RefSeq[0])) )
        for i:=0; i<len(t.AltSeq); i++ {
          if i>0 { out.Write([]byte(",")) }
          out.Write( []byte(t.AltSeq[i]) )
        }
        out.Write( []byte(fmt.Sprintf("\t%s\t%s\t%s\t%s\t%s\n", qual_field, filt_field, info_field, fmt_field, samp_field)) )



      } else {

        //DEBUG
        fmt.Printf("!!!!!!\n")

        // REF then ALT (indel)

        t_ref:=g_vcf_buffer[0]

        if t_ref.RefLen>1 {
          info_field = fmt.Sprintf("END=%d", t_ref.RefPos+t_ref.RefLen)
          out.Write( []byte(fmt.Sprintf("%s\t%d\t%s\t%c\t%s\t%s\t%s\t%s\t%s\t%s\n",
            t_ref.Chrom,
            t_ref.RefPos+1, id_field,
            t_ref.RefSeq[0], alt_field,
            qual_field, filt_field,
            info_field, fmt_field, samp_field)) )
        }

        t_alt:=g_vcf_buffer[1]

        bp_ref := t_ref.RefSeq[len(t_ref.RefSeq)-1]

        out.Write( []byte(fmt.Sprintf("%s\t%d\t%s\t%c\t", t_alt.Chrom, t_alt.RefPos, id_field, bp_ref)) )
        for i:=0; i<len(t_alt.AltSeq); i++ {
          if i>0 { out.Write([]byte(",")) }
          out.Write( []byte(string(bp_ref) + t_alt.AltSeq[i]) )
        }
        out.Write( []byte(fmt.Sprintf("\t%s\t%s\t%s\t%s\t%s\n", qual_field, filt_field, info_field, fmt_field, samp_field)) )

      }

      g_vcf_buffer = g_vcf_buffer[2:]

    } else {


      //DEBUG
      fmt.Printf("cpB\n")

      t:=g_vcf_buffer[0]


      if t.Type == pasta.REF {

        info_field = fmt.Sprintf("END=%d", t.RefPos+t.RefLen+1)
        out.Write( []byte(fmt.Sprintf("%s\t%d\t%s\t%c\t%s\t%s\t%s\t%s\t%s\t%s\n",
          t.Chrom,
          t.RefPos+1, id_field,
          t.RefSeq[0], alt_field,
          qual_field, filt_field,
          info_field, fmt_field, samp_field)) )

      } else if t.Type == pasta.ALT {

        ref_bp := byte('x')

        snp_flag := true
        if len(t.RefSeq)==1 {
          for i:=0; i<len(t.AltSeq); i++ {
            if len(t.AltSeq[i])!=1 {
              snp_flag = false
              break
            }
          }
          if snp_flag { ref_bp = t.RefSeq[0] }
        } else {
          snp_flag = false
        }

        out.Write( []byte(fmt.Sprintf("%s\t%d\t%s\t%c\t", t.Chrom, t.RefPos+1, id_field, ref_bp)) )
        for i:=0; i<len(t.AltSeq); i++ {
          if i>0 { out.Write([]byte(",")) }
          out.Write( []byte(t.AltSeq[i]) )
        }
        out.Write( []byte(fmt.Sprintf("\t%s\t%s\t%s\t%s\t%s\n", qual_field, filt_field, info_field, fmt_field, samp_field)) )

      } else if t.Type == pasta.NOC {
        filt_field = "NOCALL"
        samp_field = "./."

        info_field = fmt.Sprintf("END=%d", t.RefPos+t.RefLen+1)
        out.Write( []byte(fmt.Sprintf("%s\t%d\t%s\t%c\t%s\t%s\t%s\t%s\t%s\t%s\n",
          t.Chrom,
          t.RefPos+1, id_field,
          t.RefSeq[0], alt_field,
          qual_field, filt_field,
          info_field, fmt_field, samp_field)) )

      } else if t.Type == pasta.MSG {

        out.Write( []byte(fmt.Sprintf("msg not implemented\n")) )

      }

      g_vcf_buffer = g_vcf_buffer[1:]


    }

  }

  return nil

}

func simple_refvar_printer(vartype int, ref_start, ref_len int, refseq []byte, altseq [][]byte, info_if interface{}) error {

  info := info_if.(*RefVarInfo)

  out := os.Stdout

  chrom := info.Chrom

  if vartype == pasta.REF {

    if info.RefSeqFlag {
      out.Write( []byte(fmt.Sprintf("%s\tref\t%d\t%d\t%s\n", chrom, ref_start, ref_start+ref_len, refseq)) )
    } else {
      out.Write( []byte(fmt.Sprintf("%s\tref\t%d\t%d\t.\n", chrom, ref_start, ref_start+ref_len)) )
    }

  } else if vartype == pasta.NOC {

    if info.RefSeqFlag {

      if info.NocSeqFlag {
        out.Write( []byte(fmt.Sprintf("%s\tnoc\t%d\t%d\t%s/%s;%s\n", chrom, ref_start, ref_start+ref_len, altseq[0], altseq[1], refseq)) )
      } else {
        out.Write( []byte(fmt.Sprintf("%s\tnca\t%d\t%d\t%s/%s;%s\n", chrom, ref_start, ref_start+ref_len, altseq[0], altseq[1], refseq)) )
      }

    } else {

      if info.NocSeqFlag {
        out.Write( []byte(fmt.Sprintf("%s\tnoc\t%d\t%d\t%s/%s;.\n", chrom, ref_start, ref_start+ref_len, altseq[0], altseq[1])) )
      } else {
        out.Write( []byte(fmt.Sprintf("%s\tnoa\t%d\t%d\t.\n", chrom, ref_start, ref_start+ref_len)) )
      }
    }

  } else if vartype == pasta.ALT {

    out.Write( []byte(fmt.Sprintf("%s\talt\t%d\t%d\t%s/%s;%s\n", chrom, ref_start, ref_start+ref_len, altseq[0], altseq[1], refseq)) )

  } else if vartype == pasta.MSG {

    if info.Msg.Type == pasta.REF {
      out.Write( []byte(fmt.Sprintf("%s\tref\t%d\t%d\t.(msg)\n", chrom, ref_start, ref_start+info.Msg.N)) )
    } else if info.Msg.Type == pasta.NOC {
      out.Write( []byte(fmt.Sprintf("%s\tnoc\t%d\t%d\t.(msg)\n", chrom, ref_start, ref_start+info.Msg.N)) )
    }

  }

  return nil

}

// Read from an interleaved stream and print out a simplified variant difference format
//
// Each token from the stream should be interleaved and aligned.  Each token can be processed
// two at a time, where the first token is from the first stream and the second is from
// the second stream.  The resulting difference format spits out contigs of ref, non-ref and
// alts where appropriate.
//
// The 'process' callback will be called for every variant line that gets processed.
//
func interleave_to_diff(stream *bufio.Reader, process RefVarProcesser) error {
  alt0 := []byte{}
  alt1 := []byte{}
  refseq := []byte{}

  ref_start := 0
  ref0_len := 0
  ref1_len := 0

  stream0_pos:=0
  stream1_pos:=0

  info := RefVarInfo{}
  //info := GVCFVarInfo{}
  info.Type = pasta.BEG
  info.MessageType = pasta.BEG
  info.RefSeqFlag = gFullRefSeqFlag
  info.NocSeqFlag = gFullNocSeqFlag
  info.Out = os.Stdout
  info.Chrom = "Unk"
  //info.PrintHeader = true
  //info.Reference = "hg19"

  var bp_anchor_ref byte
  var bp_anchor_prv byte

  if g_debug { fmt.Printf("%v\n", pasta.RefDelBP) }

  curStreamState := pasta.BEG ; _ = curStreamState
  prvStreamState := pasta.BEG ; _ = prvStreamState

  var msg pasta.ControlMessage
  var prev_msg pasta.ControlMessage
  var e error

  var ch1 byte
  var e1 error

  var dbp0 int
  var dbp1 int


  for {
    is_ref0 := false
    is_ref1 := false

    is_noc0 := false
    is_noc1 := false

    message_processed_flag := false

    ch0,e0 := stream.ReadByte()
    for (e0==nil) && ((ch0=='\n') || (ch0==' ') || (ch0=='\r') || (ch0=='\t')) {
      ch0,e0 = stream.ReadByte()
    }
    if e0!=nil { break }

    if ch0=='>' {
      msg,e = pasta.ControlMessageProcess(stream)
      if e!=nil { return fmt.Errorf(fmt.Sprintf("invalid control message %v (%v)", msg, e)) }

      if (msg.Type == pasta.REF) || (msg.Type == pasta.NOC) {
        curStreamState = pasta.MSG_REF_NOC
      } else if msg.Type == pasta.CHROM {
        curStreamState = pasta.MSG_CHROM
      } else if msg.Type == pasta.POS {
        curStreamState = pasta.MSG_POS
      } else {
        //just ignore
        continue
        //return fmt.Errorf("invalid message type")
      }

      message_processed_flag = true
    }

    if !message_processed_flag {
      ch1,e1 = stream.ReadByte()
      for (e1==nil) && ((ch1=='\n') || (ch1==' ') || (ch1=='\r') || (ch1=='\t')) {
        ch1,e1 = stream.ReadByte()
      }
      if e1!=nil { break }

      stream0_pos++
      stream1_pos++

      // special case: nop
      //
      if ch0=='.' && ch1=='.' { continue }

      dbp0 = pasta.RefDelBP[ch0]
      dbp1 = pasta.RefDelBP[ch1]

      if g_debug {
        fmt.Printf("\n")
        fmt.Printf(">>> ch0 %c (%d), ch1 %c (%d), dbp0 +%d, dbp1 +%d, ref0_len %d, ref1_len %d\n", ch0, ch0, ch1, ch1, dbp0, dbp1, ref0_len, ref1_len)
      }

      if ch0=='a' || ch0=='c' || ch0=='g' || ch0=='t' {
        is_ref0 = true
      } else if ch0=='n' || ch0=='N' || ch0 == 'A' || ch0 == 'C' || ch0 == 'G' || ch0 == 'T' {
        is_noc0 = true
      }

      if ch1=='a' || ch1=='c' || ch1=='g' || ch1=='t' {
        is_ref1 = true
      } else if ch1=='n' || ch1=='N' || ch1 == 'A' || ch1 == 'C' || ch1 == 'G' || ch1 == 'T' {
        is_noc1 = true
      }

      if is_ref0 && is_ref1 {
        curStreamState = pasta.REF
      } else if is_noc0 || is_noc1 {
        curStreamState = pasta.NOC
      } else {
        curStreamState = pasta.ALT
      }

    }

    if curStreamState == pasta.BEG {

      if !is_ref0 || !is_ref1 {
        if bp,ok := pasta.RefMap[ch0] ; ok {
          refseq = append(refseq, bp)
          bp_anchor_ref = bp
        } else if bp, ok := pasta.RefMap[ch1] ; ok {
          refseq = append(refseq, bp)
          bp_anchor_ref = bp
        }
      } else if gFullRefSeqFlag {
        if bp,ok := pasta.RefMap[ch0] ; ok {
          refseq = append(refseq, bp)
          bp_anchor_ref = bp
        } else if bp, ok := pasta.RefMap[ch1] ; ok {
          refseq = append(refseq, bp)
          bp_anchor_ref = bp
        }
      }

      ref0_len+=dbp0
      ref1_len+=dbp1

      if bp_val,ok := pasta.AltMap[ch0] ; ok { alt0 = append(alt0, bp_val) }
      if bp_val,ok := pasta.AltMap[ch1] ; ok { alt1 = append(alt1, bp_val) }

      prvStreamState = curStreamState
      prev_msg = msg

      continue
    }

    if !message_processed_flag {
      if is_ref0 && is_ref1 && ch0!=ch1 {
        return fmt.Errorf(fmt.Sprintf("ERROR: stream position (%d,%d), stream0 token %c (%d), stream1 token %c (%d)",
          stream0_pos, stream1_pos, ch0, ch0, ch1, ch1))
      }
    }

    if (prvStreamState == pasta.REF) && (curStreamState != pasta.REF) {

      info.RefBP = bp_anchor_ref
      process(prvStreamState, ref_start, ref0_len, refseq, nil, &info)

      // Save the last ref BP in case the ALT is an indel.
      //
      bp_anchor_prv = '-'
      if len(refseq)>0 { bp_anchor_prv = refseq[len(refseq)-1] }

      ref_start += ref0_len

      ref0_len=0
      ref1_len=0

      alt0 = alt0[0:0]
      alt1 = alt1[0:0]
      refseq = refseq[0:0]

    } else if (prvStreamState == pasta.NOC) && (curStreamState != pasta.NOC) {

      //full_noc_flag := false
      full_noc_flag := gFullNocSeqFlag
      for ii:=0; ii<len(alt0); ii++ { if alt0[ii]!='n' { full_noc_flag = true ; break; } }
      if full_noc_flag { for ii:=0; ii<len(alt1); ii++ { if alt1[ii]!='n' { full_noc_flag = true ; break; } } }

      a0 := string(alt0)
      if len(a0) == 0 { a0 = "-" }

      a1 := string(alt1)
      if len(a1) == 0 { a1 = "-" }

      r := string(refseq)
      if len(r) == 0 { r = "-" }



      info.RefBP = bp_anchor_ref
      info.NocSeqFlag = full_noc_flag
      //process(prvStreamState, ref_start, ref0_len, refseq, [][]byte{alt0, alt1}, &info)
      process(prvStreamState, ref_start, ref0_len, []byte(r), [][]byte{[]byte(a0), []byte(a1)}, &info)

      // Save the last ref BP in case the ALT is an indel.
      //
      bp_anchor_prv = '-'
      if len(refseq)>0 { bp_anchor_prv = refseq[len(refseq)-1] }

      ref_start += ref0_len

      ref0_len=0
      ref1_len=0

      alt0 = alt0[0:0]
      alt1 = alt1[0:0]
      refseq = refseq[0:0]

    } else if (prvStreamState == pasta.ALT) && ((curStreamState == pasta.REF) || (curStreamState == pasta.NOC)) {

      a0 := string(alt0)
      if len(a0) == 0 { a0 = "-" }

      a1 := string(alt1)
      if len(a1) == 0 { a1 = "-" }

      r := string(refseq)
      if len(r) == 0 { r = "-" }

      info.RefBP = bp_anchor_prv
      process(prvStreamState, ref_start, ref0_len, []byte(r), [][]byte{[]byte(a0), []byte(a1)}, &info)

      ref_start += ref0_len

      ref0_len=0
      ref1_len=0

      alt0 = alt0[0:0]
      alt1 = alt1[0:0]
      refseq = refseq[0:0]

    //} else if prvStreamState == MSG {
    } else if prvStreamState == pasta.MSG_REF_NOC {

      info.Msg = prev_msg
      info.RefBP = bp_anchor_ref
      process(prvStreamState, ref_start, prev_msg.N, refseq, nil, &info)

      ref_start += prev_msg.N

      stream0_pos += prev_msg.N
      stream1_pos += prev_msg.N

      ref0_len=0
      ref1_len=0
      alt0 = alt0[0:0]
      alt1 = alt1[0:0]
      refseq = refseq[0:0]

    } else if prvStreamState == pasta.MSG_CHROM {
      info.Chrom = prev_msg.Chrom
    } else if prvStreamState == pasta.MSG_POS {
      ref_start = prev_msg.RefPos
    } else {
      // The current state matches the previous state.
      // Either both the current tokens are non-ref as well as the previous tokens
      // or both the current token and previous tokens are ref.
    }

    if !message_processed_flag {
      if bp_val,ok := pasta.AltMap[ch0] ; ok { alt0 = append(alt0, bp_val) }
      if bp_val,ok := pasta.AltMap[ch1] ; ok { alt1 = append(alt1, bp_val) }

      if !is_ref0 || !is_ref1 {

        //DEBUG
        //fmt.Printf("not ref (ref_start %v)... ch0 %c, ch1 %c, bp (%c,%c)\n",
        //  ref_start, ch0, ch1, pasta.RefMap[ch0], pasta.RefMap[ch1])

        if bp,ok := pasta.RefMap[ch0] ; ok {
          refseq = append(refseq, bp)
          if ref0_len==0 { bp_anchor_ref = bp }
        } else if bp, ok := pasta.RefMap[ch1] ; ok {
          refseq = append(refseq, bp)
          if ref0_len==0 { bp_anchor_ref = bp }
        }
      } else if gFullRefSeqFlag {

        if bp,ok := pasta.RefMap[ch0] ; ok {
          refseq = append(refseq, bp)
          if ref0_len==0 { bp_anchor_ref = bp }
        } else if bp, ok := pasta.RefMap[ch1] ; ok {
          refseq = append(refseq, bp)
          if ref0_len==0 { bp_anchor_ref = bp }
        }
      } else if ref0_len==0 {

        if bp,ok := pasta.RefMap[ch0] ; ok {
          if ref0_len==0 { bp_anchor_ref = bp }
        } else if bp, ok := pasta.RefMap[ch1] ; ok {
          if ref0_len==0 { bp_anchor_ref = bp }
        }
      }

      ref0_len+=dbp0
      ref1_len+=dbp1

    }

    prvStreamState = curStreamState
    prev_msg = msg

  }

  if prvStreamState == pasta.REF {

    info.RefBP = bp_anchor_ref
    process(prvStreamState, ref_start, ref0_len, refseq, [][]byte{alt0, alt1}, &info)

  } else if prvStreamState == pasta.NOC {

    //full_noc_flag := false
    full_noc_flag := gFullNocSeqFlag
    for ii:=0; ii<len(alt0); ii++ { if alt0[ii]!='n' { full_noc_flag = true ; break; } }
    if full_noc_flag { for ii:=0; ii<len(alt1); ii++ { if alt1[ii]!='n' { full_noc_flag = true ; break; } } }

    info.NocSeqFlag = full_noc_flag
    info.RefBP = bp_anchor_ref
    process(prvStreamState, ref_start, ref0_len, refseq, [][]byte{alt0, alt1}, &info)

  } else if prvStreamState == pasta.ALT {

    a0 := string(alt0)
    if len(a0) == 0 { a0 = "-" }

    a1 := string(alt1)
    if len(a1) == 0 { a1 = "-" }

    r := string(refseq)
    if len(r) == 0 { r = "-" }

    process(prvStreamState, ref_start, ref0_len, []byte(r), [][]byte{[]byte(a0), []byte(a1)}, &info)

  } else if prvStreamState == pasta.MSG_REF_NOC {

    info.Msg = prev_msg
    info.RefBP = bp_anchor_ref
    process(prvStreamState, ref_start, prev_msg.N, nil, nil, &info)

  } else if prvStreamState == pasta.MSG_CHROM {
    info.Chrom = prev_msg.Chrom
  }

  return nil
}

func pasta_to_haploid(stream *bufio.Reader, ind int) error {
  var msg pasta.ControlMessage
  var e error
  var stream0_pos int
  var dbp0 int ; _ = dbp0
  var curStreamState int ; _ = curStreamState

  out := bufio.NewWriter(os.Stdout)

  bp_count:=0
  lfmod := 50

  for {
    message_processed_flag := false

    ch0,e0 := stream.ReadByte()
    for (e0==nil) && ((ch0=='\n') || (ch0==' ') || (ch0=='\r') || (ch0=='\t')) {
      ch0,e0 = stream.ReadByte()
    }
    if e0!=nil { break }

    if ch0=='>' {
      msg,e = pasta.ControlMessageProcess(stream)
      if e!=nil { return fmt.Errorf("invalid control message") }

      if (msg.Type == pasta.REF) || (msg.Type == pasta.NOC) {
        curStreamState = pasta.MSG
      } else {
        //ignore
        continue
      }

      message_processed_flag = true
      continue
    }

    if !message_processed_flag {

      stream0_pos++

      // special case: nop
      //
      if ch0=='.' { continue }

      is_del := false ; _ = is_del
      is_ins := false ; _ = is_ins
      is_ref := false ; _ = is_ref
      is_noc := false ; _ = is_noc

      if ch0=='!' || ch0=='$' || ch0=='7' || ch0=='E' || ch0=='z' {
        is_del = true
      } else if ch0=='Q' || ch0=='S' || ch0=='W' || ch0=='d' || ch0=='Z' {
        is_ins = true
      } else if ch0=='a' || ch0=='c' || ch0=='g' || ch0=='t' {
        is_ref = true
      } else if ch0=='n' || ch0=='N' || ch0 == 'A' || ch0 == 'C' || ch0 == 'G' || ch0 == 'T' {
        is_noc = true
      }

      dbp0 = pasta.RefDelBP[ch0]

      if ind==-1 {

        // ref

        if is_ins { continue }
        if ch0 != '.' {
          out.WriteByte(pasta.RefMap[ch0])
        }

        bp_count++
        if (lfmod>0) && ((bp_count%lfmod)==0) { out.WriteByte('\n') }

      } else if ind==0 {

        // alt0

        if ch0=='.' { continue }
        if pasta.IsAltDel[ch0] { continue }

        out.WriteByte(pasta.AltMap[ch0])
        bp_count++
        if (lfmod>0) && ((bp_count%lfmod)==0) { out.WriteByte('\n') }

      }

    }

  }

  out.WriteByte('\n')
  out.Flush()

  return nil
}


func interleave_to_haploid(stream *bufio.Reader, ind int) error {
  var msg pasta.ControlMessage
  var e error
  var stream0_pos, stream1_pos int
  var dbp0,dbp1 int ; _,_ = dbp0,dbp1
  var curStreamState int ; _ = curStreamState

  out := bufio.NewWriter(os.Stdout)

  bp_count:=0
  lfmod := 50

  for {
    message_processed_flag := false

    var ch1 byte
    var e1 error

    ch0,e0 := stream.ReadByte()
    for (e0==nil) && ((ch0=='\n') || (ch0==' ') || (ch0=='\r') || (ch0=='\t')) {
      ch0,e0 = stream.ReadByte()
    }
    if e0!=nil { break }

    if ch0=='>' {
      msg,e = pasta.ControlMessageProcess(stream)
      if e!=nil { return fmt.Errorf("invalid control message") }

      if (msg.Type == pasta.REF) || (msg.Type == pasta.NOC) {
        curStreamState = pasta.MSG
      } else {
        //ignore
        continue
        //return fmt.Errorf("invalid message type")
      }

      message_processed_flag = true
      continue
    }

    //fmt.Printf("??? ch0 %c\n", ch0)

    if !message_processed_flag {
      ch1,e1 = stream.ReadByte()
      for (e1==nil) && ((ch1=='\n') || (ch1==' ') || (ch1=='\r') || (ch1=='\t')) {
        ch1,e1 = stream.ReadByte()
      }
      if e1!=nil { break }

      //fmt.Printf("??? ch1 %c\n", ch1)

      stream0_pos++
      stream1_pos++

      // special case: nop
      //
      if ch0=='.' && ch1=='.' { continue }

      dbp0 = pasta.RefDelBP[ch0]
      dbp1 = pasta.RefDelBP[ch1]

      anch_bp := ch0
      if anch_bp == '.' { anch_bp = ch1 }

      is_del := []bool{false,false}
      is_ins := []bool{false,false}
      is_ref := []bool{false,false} ; _ = is_ref
      is_noc := []bool{false,false} ; _ = is_noc

      if ch0=='!' || ch0=='$' || ch0=='7' || ch0=='E' || ch0=='z' {
        is_del[0] = true
      } else if ch0=='Q' || ch0=='S' || ch0=='W' || ch0=='d' || ch0=='Z' {
        is_ins[0] = true
      } else if ch0=='a' || ch0=='c' || ch0=='g' || ch0=='t' {
        is_ref[0] = true
      } else if ch0=='n' || ch0=='N' || ch0 == 'A' || ch0 == 'C' || ch0 == 'G' || ch0 == 'T' {
        is_noc[0] = true
      }

      if ch1=='!' || ch1=='$' || ch1=='7' || ch1=='E' || ch1=='z' {
        is_del[1] = true
      } else if ch1=='Q' || ch1=='S' || ch1=='W' || ch1=='d' || ch1=='Z' {
        is_ins[1] = true
      } else if ch1=='a' || ch1=='c' || ch1=='g' || ch1=='t' {
        is_ref[1] = true
      } else if ch1=='n' || ch1=='N' || ch1 == 'A' || ch1 == 'C' || ch1 == 'G' || ch1 == 'T' {
        is_noc[1] = true
      }

      /*
      if (is_del[0] && (!is_del[1] && ch1!='.')) ||
         (is_del[1] && (!is_del[0] && ch0!='.')) {
        return fmt.Errorf( fmt.Sprintf("deletion mismatch (ch %c,%c (%v,%v) @ %v)", ch0, ch1, ch0, ch1, bp_count) )
      }
      */

      if (is_ins[0] && (!is_ins[1] && ch1!='.')) ||
         (is_ins[1] && (!is_ins[0] && ch0!='.')) {
        return fmt.Errorf( fmt.Sprintf("insertion mismatch (ch %c,%c ord(%v,%v) @ %v)", ch0, ch1, ch0, ch1, bp_count) )
      }

      if ind==-1 {

        // ref

        if is_ins[0] || is_ins[1] { continue }
        if ch0 != '.' {
          //fmt.Printf("%c", pasta.RefMap[ch0])
          out.WriteByte(pasta.RefMap[ch0])
        } else {
          //fmt.Printf("%c", pasta.RefMap[ch1])
          out.WriteByte(pasta.RefMap[ch1])
        }

        bp_count++
        //if (lfmod>0) && ((bp_count%lfmod)==0) { fmt.Printf("\n") }
        if (lfmod>0) && ((bp_count%lfmod)==0) { out.WriteByte('\n') }

      } else if ind==0 {

        // alt0

        if ch0=='.' { continue }
        if pasta.IsAltDel[ch0] { continue }

        //fmt.Printf("%c", pasta.AltMap[ch0])
        out.WriteByte(pasta.AltMap[ch0])
        bp_count++
        //if (lfmod>0) && ((bp_count%lfmod)==0) { fmt.Printf("\n") }
        if (lfmod>0) && ((bp_count%lfmod)==0) { out.WriteByte('\n') }

      } else if ind==1 {

        // alt1

        if ch1=='.' { continue }
        if pasta.IsAltDel[ch1] { continue }

        //fmt.Printf("%c", pasta.AltMap[ch1])
        out.WriteByte(pasta.AltMap[ch1])
        bp_count++
        //if (lfmod>0) && ((bp_count%lfmod)==0) { fmt.Printf("\n") }
        if (lfmod>0) && ((bp_count%lfmod)==0) { out.WriteByte('\n') }

      }

    }


  }

  //fmt.Printf("\n")
  out.WriteByte('\n')
  out.Flush()

  return nil

}

/*
func InterleaveStreams(stream_A, stream_B io.Reader, w io.Writer) error {
  var e0, e1 error
  ref_pos := [2]int{0,0}
  stm_pos := [2]int{0,0} ; _ = stm_pos
  ch_val := [2]byte{0,0}
  dot := [1]byte{'.'}

  stream_a := bufio.NewReader(stream_A)
  stream_b := bufio.NewReader(stream_B)
  out := bufio.NewWriter(w)

  for {

    if ref_pos[0] == ref_pos[1] {

      ch_val[0],e0 = stream_a.ReadByte()
      ch_val[1],e1 = stream_b.ReadByte()

      stm_pos[0]++
      stm_pos[1]++
    } else if ref_pos[0] < ref_pos[1] {
      ch_val[0],e0 = stream_a.ReadByte()

      stm_pos[0]++
    } else if ref_pos[0] > ref_pos[1] {
      ch_val[1],e1 = stream_b.ReadByte()

      stm_pos[1]++
    }

    if e0!=nil && e1!=nil { break }

    if ch_val[0] == '.' && ch_val[1] == '.' { continue }
    if ref_pos[0] == ref_pos[1] {

      if (ch_val[0]!='Q') && (ch_val[0]!='S') && (ch_val[0]!='W') && (ch_val[0]!='d') && (ch_val[0]!='.') && (ch_val[0]!='\n') && (ch_val[0]!=' ') {
        ref_pos[0]++
      }

      if (ch_val[1]!='Q') && (ch_val[1]!='S') && (ch_val[1]!='W') && (ch_val[1]!='d') && (ch_val[1]!='.') && (ch_val[1]!='\n') && (ch_val[1]!=' ') {
        ref_pos[1]++
      }

    } else if ref_pos[0] < ref_pos[1] {
      if (ch_val[0]!='Q') && (ch_val[0]!='S') && (ch_val[0]!='W') && (ch_val[0]!='d') && (ch_val[0]!='.') && (ch_val[0]!='\n') && (ch_val[0]!=' ') {
        ref_pos[0]++
      }
    } else if ref_pos[0] > ref_pos[1] {

      if (ch_val[1]!='Q') && (ch_val[1]!='S') && (ch_val[1]!='W') && (ch_val[1]!='d') && (ch_val[1]!='.') && (ch_val[1]!='\n') && (ch_val[1]!=' ') {
        ref_pos[1]++
      }
    }

    if ref_pos[0]==ref_pos[1] {
      out.WriteByte(ch_val[0])
      out.WriteByte(ch_val[1])
    } else if ref_pos[0] < ref_pos[1] {
      out.WriteByte(ch_val[0])
      out.WriteByte(dot[0])
    } else if ref_pos[0] > ref_pos[1] {
      out.WriteByte(dot[0])
      out.WriteByte(ch_val[1])
    }

  }

  return nil
}

func WriteVarDiff(vardiff []VarDiff, w io.Writer) {

  for i:=0; i<len(vardiff); i++ {
    if vardiff[i].Type == "REF" {

      if gFullRefSeqFlag {
        r:=vardiff[i].RefSeq
        if len(vardiff[i].RefSeq)==0 { r="-" }
        fmt.Printf("ref\t%d\t%d\t%s\n",
          vardiff[i].RefStart, vardiff[i].RefStart + vardiff[i].RefLen, r)
      } else {
        fmt.Printf("ref\t%d\t%d\t.\n",
          vardiff[i].RefStart, vardiff[i].RefStart + vardiff[i].RefLen)
      }
    } else if vardiff[i].Type == "ALT" {
      a0 := vardiff[i].AltSeq[0]
      if len(a0)==0 { a0 = "-" }
      a1 := vardiff[i].AltSeq[1]
      if len(a1)==0 { a1 = "-" }
      r := vardiff[i].RefSeq
      if len(r)==0 { r="-" }
      fmt.Printf("alt\t%d\t%d\t%s/%s;%s\n",
        vardiff[i].RefStart, vardiff[i].RefStart + vardiff[i].RefLen,
        a0, a1, r)
    }
  }
}
*/


func diff_to_interleave(ain *autoio.AutoioHandle) {

  n_allele := 2
  lfmod := 50
  //lfmod := -1
  bp_count := 0

  chrom := ""
  pos := -1

  first_pass := true

  for ain.ReadScan() {
    l := ain.ReadText()

    if len(l)==0 || l=="" { continue }

    diff_parts := strings.Split(l, "\t")

    chrom_s := diff_parts[0]
    type_s := diff_parts[1]
    st_s := diff_parts[2] ; _ = st_s
    en_s := diff_parts[3] ; _ = en_s
    field := diff_parts[4]

    control_message := false

    if chrom != chrom_s {

      if !first_pass && !control_message { fmt.Printf("\n") }

      fmt.Printf(">C{%s}", chrom_s)
      chrom = chrom_s

      control_message = true
    }

    _st,e := strconv.ParseUint(st_s, 10, 64)
    if e==nil {

      if pos != int(_st) {
        if !first_pass && !control_message { fmt.Printf("\n") }
        fmt.Printf(">P{%d}", _st)
        pos = int(_st)

        control_message = true
      }

    }

    if control_message { fmt.Printf("\n") }
    first_pass = false

    if type_s == "ref" {

      for i:=0; i<len(field); i++ {
        for a:=0; a<n_allele; a++ {
          fmt.Printf("%c", field[i])

          bp_count++
          if (lfmod>0) && ((bp_count%lfmod)==0) {
            fmt.Printf("\n")
          }
        }
      }

      pos += len(field)

    } else if type_s == "alt" || type_s == "nca"  || type_s == "noc" {

      field_parts := strings.Split(field, ";")
      alt_parts := strings.Split(field_parts[0], "/")
      if len(alt_parts)==1 { alt_parts = append(alt_parts, alt_parts[0]) }
      refseq := field_parts[1]

      mM := len(alt_parts[0])
      if len(alt_parts[1]) > mM { mM = len(alt_parts[1]) }
      if len(refseq) > mM { mM = len(refseq) }

      for i:=0; i<mM; i++  {

        for a:=0; a<len(alt_parts); a++ {

          if i<len(alt_parts[a]) {
            if i<len(refseq) {
              fmt.Printf("%c", pasta.SubMap[refseq[i]][alt_parts[a][i]])
            } else {
              fmt.Printf("%c", pasta.InsMap[alt_parts[a][i]])
            }
          } else if i<len(refseq) {
            fmt.Printf("%c", pasta.DelMap[refseq[i]])
          } else {
            fmt.Printf(".")
          }

          bp_count++
          if (lfmod>0) && ((bp_count%lfmod)==0) {
            fmt.Printf("\n")
          }

        }

      }

      if refseq != "-" {
        pos += len(refseq)
      }

    }

  }

  fmt.Printf("\n")

}

func _main_diff_to_rotini( c *cli.Context ) {
  infn_slice := c.StringSlice("input")
  if len(infn_slice)<1 {
    //fmt.Fprintf(os.Stderr, "provide input file")
    //os.Exit(1)

    infn_slice = append(infn_slice, "-")
  }

  ain,err := autoio.OpenReadScanner(infn_slice[0])
  if err!=nil {
    fmt.Fprintf(os.Stderr, "%v", err)
    os.Exit(1)
  }
  defer ain.Close()

  diff_to_interleave(&ain)

}

func _main_gvcf_to_rotini(c *cli.Context) {
  var e error

  infn_slice := c.StringSlice("input")
  if len(infn_slice)<1 {
    infn_slice = append(infn_slice, "-")
  }

  ain,err := autoio.OpenReadScanner(infn_slice[0])
  if err!=nil {
    fmt.Fprintf(os.Stderr, "%v", err)
    os.Exit(1)
  }
  defer ain.Close()

  fp := os.Stdin
  if c.String("refstream")!="-" {
    fp,e = os.Open(c.String("refstream"))
    if e!=nil {
      fmt.Fprintf(os.Stderr, "%v", e)
      os.Exit(1)
    }
    defer fp.Close()
  }
  ref_stream := bufio.NewReader(fp)

  out := bufio.NewWriter(os.Stdout)

  gvcf := GVCFRefVar{}
  gvcf.Init()

  line_no:=0
  gvcf.PastaBegin(out)
  for ain.ReadScan() {
    gvcf_line := ain.ReadText()
    line_no++

    if len(gvcf_line)==0 || gvcf_line=="" { continue }
    e:=gvcf.Pasta(gvcf_line, ref_stream, out)
    if e!=nil { fmt.Fprintf(os.Stderr, "ERROR: %v at line %v\n", e, line_no); return }
  }
  gvcf.PastaEnd(out)

  out.Flush()

}

func _main_gff_to_rotini(c *cli.Context) {
  var e error

  infn_slice := c.StringSlice("input")
  if len(infn_slice)<1 {
    infn_slice = append(infn_slice, "-")
  }

  ain,err := autoio.OpenReadScanner(infn_slice[0])
  if err!=nil {
    fmt.Fprintf(os.Stderr, "%v", err)
    os.Exit(1)
  }
  defer ain.Close()

  fp := os.Stdin
  if c.String("refstream")!="-" {
    fp,e = os.Open(c.String("refstream"))
    if e!=nil {
      fmt.Fprintf(os.Stderr, "%v", e)
      os.Exit(1)
    }
    defer fp.Close()
  }
  ref_stream := bufio.NewReader(fp)

  out := bufio.NewWriter(os.Stdout)

  gff := GFFRefVar{}
  gff.Init()

  if c.Int("start") > 0 {
    gff.RefPos = c.Int("start")
    gff.PrevRefPos = gff.RefPos
  }

  line_no:=0
  gff.PastaBegin(out)
  for ain.ReadScan() {
    gff_line := ain.ReadText()
    line_no++

    if len(gff_line)==0 || gff_line=="" { continue }
    e:=gff.Pasta(gff_line, ref_stream, out)
    if e!=nil { fmt.Fprintf(os.Stderr, "ERROR: %v at line %v\n", e, line_no); return }
  }

  gff.PastaRefEnd(ref_stream, out)
  gff.PastaEnd(out)

}

func _main_cgivar_to_rotini(c *cli.Context) {
  var e error

  infn_slice := c.StringSlice("input")
  if len(infn_slice)<1 {
    infn_slice = append(infn_slice, "-")
  }

  ain,err := autoio.OpenReadScanner(infn_slice[0])
  if err!=nil {
    fmt.Fprintf(os.Stderr, "%v", err)
    os.Exit(1)
  }
  defer ain.Close()

  fp := os.Stdin
  if c.String("refstream")!="-" {
    fp,e = os.Open(c.String("refstream"))
    if e!=nil {
      fmt.Fprintf(os.Stderr, "%v", e)
      os.Exit(1)
    }
    defer fp.Close()
  }
  ref_stream := bufio.NewReader(fp)

  out := bufio.NewWriter(os.Stdout)

  cgivar := CGIRefVar{}
  cgivar.Init()

  line_no:=0
  cgivar.PastaBegin(out)
  for ain.ReadScan() {
    cgivar_line := ain.ReadText()
    line_no++

    if len(cgivar_line)==0 || cgivar_line=="" { continue }
    e:=cgivar.Pasta(cgivar_line, ref_stream, out)
    if e!=nil { fmt.Fprintf(os.Stderr, "ERROR: %v at line %v\n", e, line_no); return }
  }
  cgivar.PastaEnd(out)

}


func _main_cgivar_to_pasta(c *cli.Context) {
  var e error

  infn_slice := c.StringSlice("input")
  if len(infn_slice)<1 {
    infn_slice = append(infn_slice, "-")
  }

  ain,err := autoio.OpenReadScanner(infn_slice[0])
  if err!=nil {
    fmt.Fprintf(os.Stderr, "%v", err)
    os.Exit(1)
  }
  defer ain.Close()

  fp := os.Stdin
  if c.String("refstream")!="-" {
    fp,e = os.Open(c.String("refstream"))
    if e!=nil {
      fmt.Fprintf(os.Stderr, "%v", e)
      os.Exit(1)
    }
    defer fp.Close()
  }
  ref_stream := bufio.NewReader(fp)

  out := bufio.NewWriter(os.Stdout)

  cgivar := CGIRefVar{}
  cgivar.Init()
  cgivar.Ploidy=1

  line_no:=0
  cgivar.PastaBegin(out)
  for ain.ReadScan() {
    cgivar_line := ain.ReadText()
    line_no++

    if len(cgivar_line)==0 || cgivar_line=="" { continue }
    e:=cgivar.Pasta(cgivar_line, ref_stream, out)
    if e!=nil { fmt.Fprintf(os.Stderr, "ERROR: %v at line %v\n", e, line_no); return }
  }
  cgivar.PastaEnd(out)

}

func _main_fasta_to_pasta(c *cli.Context) {

  var e error

  infn_slice := c.StringSlice("input")
  if len(infn_slice)<1 {
    infn_slice = append(infn_slice, "-")
  }

  ain,err := autoio.OpenReadScanner(infn_slice[0])
  if err!=nil {
    fmt.Fprintf(os.Stderr, "%v", err)
    os.Exit(1)
  }
  defer ain.Close()

  fp := os.Stdin
  if c.String("refstream")!="-" {
    fp,e = os.Open(c.String("refstream"))
    if e!=nil {
      fmt.Fprintf(os.Stderr, "%v", e)
      os.Exit(1)
    }
    defer fp.Close()
  }
  ref_stream := bufio.NewReader(fp)

  out := bufio.NewWriter(os.Stdout)

  fi := FASTAInfo{}
  fi.Init()
  fi.Allele=0

  line_no:=0
  fi.PastaBegin(out)
  for ain.ReadScan() {
    fasta_line := ain.ReadText()
    line_no++

    if len(fasta_line)==0 || fasta_line=="" { continue }
    e:=fi.Pasta(fasta_line, ref_stream, out)
    if e!=nil { fmt.Fprintf(os.Stderr, "ERROR: %v at line %v\n", e, line_no); return }
  }
  fi.PastaEnd(out)


}


func _main( c *cli.Context ) {
  var e error
  action := "echo"

  msg_slice := c.StringSlice("Message")
  msg_str := ""
  for i:=0; i<len(msg_slice); i++ {
    msg_str += ">" + msg_slice[i]
  }

  if c.String("action") != "" { action = c.String("action") }

  if action == "diff-rotini" {
    _main_diff_to_rotini(c)
    return
  } else if action == "gff-rotini" {
    _main_gff_to_rotini(c)
    return
  } else if action == "gvcf-rotini" {
    _main_gvcf_to_rotini(c)
    return
  } else if action == "cgivar-pasta" {
    _main_cgivar_to_pasta(c)
    return
  } else if action == "cgivar-rotini" {
    _main_cgivar_to_rotini(c)
    return
  } else if action == "fasta-pasta" {
    _main_fasta_to_pasta(c)
    return
  }


  infn_slice := c.StringSlice("input")

  var stream *bufio.Reader
  var stream_b *bufio.Reader

  g_debug = c.Bool("debug")

  gFullRefSeqFlag = c.Bool("full-sequence")
  gFullNocSeqFlag = c.Bool("full-nocall-sequence")

  n_inp_stream := 0

  if len(infn_slice)>0 {
    fp := os.Stdin
    if infn_slice[0]!="-" {
      fp,e = os.Open(infn_slice[0])
      if e!=nil {
        fmt.Fprintf(os.Stderr, "%v", e)
        os.Exit(1)
      }
      defer fp.Close()
    }
    stream = bufio.NewReader(fp)

    n_inp_stream++

    /*
  } else {
    fmt.Fprintf(os.Stderr, "Provide input stream")
    cli.ShowAppHelp(c)

    os.Exit(1)
    */

  }

  if len(infn_slice)>1 {
    fp,e := os.Open(infn_slice[1])
    if e!=nil {
      fmt.Fprintf(os.Stderr, "%v", e)
      os.Exit(1)
    }
    defer fp.Close()
    stream_b = bufio.NewReader(fp)

    n_inp_stream++

    action = "interleave"
  }

  if c.String("action") != "" { action = c.String("action") }

  aout,err := autoio.CreateWriter( c.String("output") ) ; _ = aout
  if err!=nil {
    fmt.Fprintf(os.Stderr, "%v", err)
    os.Exit(1)
  }
  defer func() { aout.Flush() ; aout.Close() }()

  if c.Bool( "pprof" ) {
    gProfileFlag = true
    gProfileFile = c.String("pprof-file")
  }

  if c.Bool( "mprof" ) {
    gMemProfileFlag = true
    gMemProfileFile = c.String("mprof-file")
  }

  gVerboseFlag = c.Bool("Verbose")

  if c.Int("max-procs") > 0 {
    runtime.GOMAXPROCS( c.Int("max-procs") )
  }

  if gProfileFlag {
    prof_f,err := os.Create( gProfileFile )
    if err != nil {
      fmt.Fprintf( os.Stderr, "Could not open profile file %s: %v\n", gProfileFile, err )
      os.Exit(2)
    }

    pprof.StartCPUProfile( prof_f )
    defer pprof.StopCPUProfile()
  }

  if (action!="ref-rstream") && (action != "rstream") && (n_inp_stream==0) {

    if action=="interleave" {
      fmt.Fprintf(os.Stderr, "Provide input stream")
      cli.ShowAppHelp(c)
      os.Exit(1)
    }

    stream = bufio.NewReader(os.Stdin)
  }

  //---

  if action == "echo" {
    echo_stream(stream)
  } else if action == "filter-pasta" {
    out := bufio.NewWriter(os.Stdout)
    pasta_filter(stream, out, c.Int("start"), c.Int("n"))
    out.Flush()
  } else if action == "filter-rotini" {
    out := bufio.NewWriter(os.Stdout)
    interleave_filter(stream, out, c.Int("start"), c.Int("n"))
    out.Flush()
  } else if action == "interleave" {

    //interleave_streams(stream, stream_b, os.Stdout)
    pasta.InterleaveStreams(stream, stream_b, os.Stdout)

  } else if action == "ref-rstream" {

    r_ctx := random_stream_context_from_param( c.String("param") )
    random_ref_stream(r_ctx)

  } else if action == "rstream" {

    r_ctx := random_stream_context_from_param( c.String("param") )
    random_stream(r_ctx)

    //FASTA
  } else if action == "pasta-fasta" {

    fi := FASTAInfo{}
    fi.Init()

    out := bufio.NewWriter(os.Stdout)
    fi.Header(out)
    e := fi.Stream(stream, out)
    if e!=nil {
      fmt.Fprintf(os.Stderr, "\nERROR: %v\n", e)
      os.Exit(1)
    }
    fi.PrintEnd(out)

  } else if action == "diff-rotini" {

    //e:=diff_to_interleave(&stream)
    //if e!=nil { fmt.Fprintf(os.Stderr, "%v\n", e); return }

  } else if action == "rotini-diff" {

    e:=interleave_to_diff(stream, simple_refvar_printer)
    if e!=nil { fmt.Fprintf(os.Stderr, "%v\n", e) ; return }
  } else if action == "rotini" {
  } else if action == "pasta-ref" {
    e := pasta_to_haploid(stream, -1)
    if e!=nil {
      fmt.Fprintf(os.Stderr, "\nERROR: %v\n", e)
      os.Exit(1)
    }
  } else if action == "rotini-ref" {
    e := interleave_to_haploid(stream, -1)
    if e!=nil {
      fmt.Fprintf(os.Stderr, "\nERROR: %v\n", e)
      os.Exit(1)
    }
  } else if action == "rotini-alt0" {
    interleave_to_haploid(stream, 0)
  } else if action == "rotini-alt1" {
    interleave_to_haploid(stream, 1)
  } else if action == "rotini-gff" {

    gff := GFFRefVar{}
    gff.Init()

    e:=interleave_to_diff_iface(stream, &gff, os.Stdout)
    if e!=nil { fmt.Fprintf(os.Stderr, "%v\n", e) ; return }

  } else if action == "rotini-gvcf" {

    gvcf := GVCFRefVar{}
    gvcf.Init()

    // We need the full reference sequence for beginning and ending bases
    //
    gFullRefSeqFlag = true

    //e:=interleave_to_diff(stream, gvcf_refvar_printer)
    e:=interleave_to_diff_iface(stream, &gvcf, os.Stdout)
    if e!=nil { fmt.Fprintf(os.Stderr, "%v\n", e) ; return }

  } else if action == "rotini-cgivar" {

    cgivar := CGIRefVar{}
    cgivar.Init()

    //e:=interleave_to_diff(stream, gvcf_refvar_printer)
    e:=interleave_to_diff_iface(stream, &cgivar, os.Stdout)
    if e!=nil { fmt.Fprintf(os.Stderr, "%v\n", e) ; return }

  } else if action == "fastj-rotini" {

    //
    // FastJ to rotini
    //

    fp := os.Stdin
    if c.String("refstream")!="-" {
      fp,e = os.Open(c.String("refstream"))
      if e!=nil {
        fmt.Fprintf(os.Stderr, "%v", e)
        os.Exit(1)
      }
      defer fp.Close()
    }
    ref_stream := bufio.NewReader(fp)

    assembly_fp,e := os.Open(c.String("assembly"))
    if e!=nil {
      fmt.Fprintf(os.Stderr, "%v", e)
      os.Exit(1)
    }
    defer assembly_fp.Close()
    assembly_stream := bufio.NewReader(assembly_fp)

    out := bufio.NewWriter(os.Stdout)

    fji := FastJInfo{}

    e = fji.Pasta(stream, ref_stream, assembly_stream, out)
    if e!=nil {
      fmt.Fprintf(os.Stderr, "%v\n", e)
      os.Exit(1)
    }

  } else if action == "rotini-fastj" {

    //
    // rotini to FastJ
    //

    tag_fp,e := os.Open(c.String("tag"))
    if e!=nil {
      fmt.Fprintf(os.Stderr, "%v", e)
      os.Exit(1)
    }
    defer tag_fp.Close()

    assembly_fp,e := os.Open(c.String("assembly"))
    if e!=nil {
      fmt.Fprintf(os.Stderr, "%v", e)
      os.Exit(1)
    }
    defer assembly_fp.Close()

    tag_reader := bufio.NewReader(tag_fp)
    assembly_reader := bufio.NewReader(assembly_fp)

    fji := FastJInfo{}
    fji.RefPos = c.Int("start")
    fji.RefBuild = c.String("build")
    fji.Chrom = c.String("chrom")

    _tilepath,e := strconv.ParseUint(c.String("tilepath"), 16, 64)
    if e!=nil {
      fmt.Fprintf(os.Stderr, "%v", e)
      os.Exit(1)
    }
    fji.TagPath = int(_tilepath)

    out := bufio.NewWriter(os.Stdout)

    err := fji.Convert(stream, tag_reader, assembly_reader, out)
    if err!=nil {
      fmt.Fprintf(os.Stderr, "%v",err)
      os.Exit(1)
    }

  } else {
    fmt.Printf("invalid action\n")
    os.Exit(1)
  }

}

func main() {

  app := cli.NewApp()
  app.Name  = "pasta"
  app.Usage = "pasta"
  app.Version = VERSION_STR
  app.Author = "Curoverse, Inc."
  app.Email = "info@curoverse.com"
  app.Action = func( c *cli.Context ) { _main(c) }

  app.Flags = []cli.Flag{
    cli.StringSliceFlag{
      Name: "input, i",
      Usage: "INPUT",
    },

    cli.StringFlag{
      Name: "output, o",
      Value: "-",
      Usage: "OUTPUT",
    },

    cli.StringFlag{
      Name: "refstream, r",
      Value: "-",
      Usage: "Reference stream (lower case)",
    },

    cli.StringFlag{
      Name: "action, a",
      Usage: "Action: rstream, ref-rstream, rotini-(diff|gvcf|gff|cgivar|fastj|ref|alt0|alt1), (diff|gvcf|cgivar|fastj)-rotini, interleave, echo",
    },

    cli.StringFlag{
      Name: "tag, T",
      Usage: "Tag input",
    },

    cli.StringFlag{
      Name: "assembly, A",
      Usage: "Assembly input",
    },

    cli.StringFlag{
      Name: "tilepath",
      Value: "0",
      Usage: "Tile path name, in hex (e.g. 2fa)",
    },

    cli.StringFlag{
      Name: "param, p",
      Usage: "Parameter",
    },

    cli.StringFlag{
      Name: "build",
      Usage: "e.g. hg19",
    },

    cli.StringFlag{
      Name: "chrom",
      Usage: "e.g. chr12",
    },

    cli.IntFlag{
      Name: "start, s",
      Usage: "Reference start",
    },

    cli.IntFlag{
      Name: "len, n",
      Usage: "Length",
    },

    cli.BoolFlag{
      Name: "debug, d",
      Usage: "Debug",
    },

    cli.BoolFlag{
      Name: "full-sequence, F",
      Usage: "Display full sequence",
    },

    cli.BoolFlag{
      Name: "full-nocall-sequence",
      Usage: "Display full nocall sequence",
    },

    cli.IntFlag{
      Name: "max-procs, N",
      Value: -1,
      Usage: "MAXPROCS",
    },

    cli.BoolFlag{
      Name: "Verbose, V",
      Usage: "Verbose flag",
    },

    cli.BoolFlag{
      Name: "pprof",
      Usage: "Profile usage",
    },

    cli.StringFlag{
      Name: "pprof-file",
      Value: gProfileFile,
      Usage: "Profile File",
    },

    cli.StringSliceFlag{
      Name: "Message, M",
      Usage: "Add message to stream",
    },

    cli.BoolFlag{
      Name: "mprof",
      Usage: "Profile memory usage",
    },

    cli.StringFlag{
      Name: "mprof-file",
      Value: gMemProfileFile,
      Usage: "Profile Memory File",
    },

  }

  app.Run( os.Args )

  if gMemProfileFlag {
    fmem,err := os.Create( gMemProfileFile )
    if err!=nil { panic(fmem) }
    pprof.WriteHeapProfile(fmem)
    fmem.Close()
  }

}
