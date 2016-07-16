package gvcf

import "fmt"
import "strconv"
import "strings"
import "bufio"
//import "os"
import "io"
import "bytes"

import "time"

import "github.com/abeconnelly/pasta"

type GVCFRefVarInfo struct {
  chrom string

  refseq string
  altseq []string
  vartype int
  ref_start int
  ref_len int
}

type GVCFRefVar struct {
  Type int
  MessageType int
  RefSeqFlag bool
  NocSeqFlag bool
  Out io.Writer
  Msg pasta.ControlMessage
  RefBP byte
  Allele int

  ChromStr string
  RefPos int

  OCounter int
  LFMod int

  PrintHeader bool
  Reference string
  DataSource string

  PrevRefBPStart byte
  PrevRefBPEnd byte

  VCFVer string

  Date time.Time

  Id string
  Qual string
  Filter string
  Info string
  Format string

  PrevStartRefBase byte
  PrevEndRefBase byte
  PrevStartRefPos int
  PrevRefLen int
  PrevVarType int

  State int

  StateHistory []GVCFRefVarInfo

  BeginningAltCondition bool
}

func (g *GVCFRefVar) Init() {
  g.PrintHeader = true
  g.DataSource = "unknown"
  g.Reference = "unknown"

  g.ChromStr = "Unk"
  g.RefPos = 0
  g.Allele = 2

  g.OCounter = 0
  g.LFMod = 50

  g.VCFVer = "VCFv4.1"
  g.Date = time.Now()

  g.Id = "."
  g.Qual = "."
  g.Filter = ""
  g.Info = ""
  g.Format = "GT"

  g.BeginningAltCondition = false

  g.State = pasta.BEG
}

func (g *GVCFRefVar) Chrom(chr string) { g.ChromStr = chr }
func (g *GVCFRefVar) Pos(pos int) { g.RefPos = pos }
func (g *GVCFRefVar) GetRefPos() int { return g.RefPos }
func (g *GVCFRefVar) Header(out *bufio.Writer) error {

  hdr := []string{};
  hdr = append(hdr, fmt.Sprintf("##fileformat=%s", g.VCFVer))
  hdr = append(hdr, fmt.Sprintf("##fileDate=%d%02d%02d", g.Date.Year(), g.Date.Month(), g.Date.Day()))
  hdr = append(hdr, fmt.Sprintf("##source=\"%s\"", g.DataSource))
  hdr = append(hdr, fmt.Sprintf("##reference=\"%s\"", g.Reference))
  hdr = append(hdr, "##FILTER=<ID=NOCALL,Description=\"Some or all of this record had no sequence calls\">")
  hdr = append(hdr, "##FORMAT=<ID=GT,Number=1,Type=String,Description=\"Genotype\">")
  hdr = append(hdr, "##INFO=<ID=END,Number=1,Type=Integer,Description=\"Stop position of the interval\">")
  hdr = append(hdr, "#CHROM\tPOS\tID\tREF\tALT\tQUAL\tFILTER\tINFO\tFORMAT\tSAMPLE")

  out.WriteString( strings.Join(hdr, "\n") + "\n" )

  return nil
}

//---

// 0      1     2   3   4   5    6      7    8      9
// chrom  pos   id  ref alt qual filter info format sample
//
func (g *GVCFRefVar) EmitLine(vartype int, vcf_ref_pos, vcf_ref_len int, vcf_ref_base byte, alt_field string, sample_field string, out *bufio.Writer) error {

  //info_field := fmt.Sprintf("END=%d", vcf_ref_pos+vcf_ref_len)
  info_field := fmt.Sprintf("END=%d", vcf_ref_pos+vcf_ref_len-1)

  //DEBUG
  fmt.Printf("emitline: vartype %v, chrom %s, vcf_ref_pos %v+%v, id %v, vcf_ref_base %v, info_field %v, sample_field %v\n",
    vartype, g.ChromStr, vcf_ref_pos, vcf_ref_len, g.Id, vcf_ref_base, info_field, sample_field)

  return nil
}

func (g *GVCFRefVar) _construct_alt_field() string {
  return ""
}

func (g *GVCFRefVar) _construct_sample_field() string {
  return ""
}

// return reference string, array of alt strings (unique) and the gt string (e.g. "0/0")
//
func (g *GVCFRefVar) _ref_alt_gt_fields(refseq string, altseq []string) (string,[]string,string) {
  _allele_n := 0

  _refseq := ""
  if len(refseq)>0 && refseq[0]!='-' {
    _refseq = string(refseq)
  }

  // Find unique altseqs (take out '-' if present)
  //

  gt_idx_str := []string{}
  gt_idx := []int{}

  altseq_uniq := []string{}
  _set := make(map[string]int)
  _set[_refseq] = _allele_n
  _allele_n++
  var ts string
  var idx int

  if len(altseq) < g.Allele {
    n := g.Allele - len(altseq)
    for ii:=0; ii<n; ii++ {
      altseq = append(altseq, altseq[0])
    }
  }

  for ii:=0; ii<len(altseq); ii++ {
    if len(altseq[ii])==0 || altseq[ii][0] == '-' {
      ts = ""
    } else {
      ts = string(altseq[ii])
    }

    if _,ok := _set[ts] ; !ok {
      _set[ts] = _allele_n
      _allele_n++
      altseq_uniq = append(altseq_uniq, ts)
    }

    idx,_ = _set[ts]
    gt_idx_str = append(gt_idx_str, fmt.Sprintf("%d", idx))
    gt_idx = append(gt_idx, idx)
  }

  gt_field := strings.Join(gt_idx_str, "/") ; _ = gt_field

  return _refseq, altseq_uniq, gt_field
}



func (g *GVCFRefVar) _emit_alt_left_anchor(info GVCFRefVarInfo, out *bufio.Writer) {

  a_refseq,a_alt,a_gt_field := g._ref_alt_gt_fields(info.refseq, info.altseq)
  _ = a_alt

  a_start := info.ref_start+1
  a_len := info.ref_len

  alt_field := strings.Join(a_alt, ",")

  a_ref_bp := byte('.')
  if len(a_refseq)>0 { a_ref_bp = a_refseq[0] }

  a_filt_field := "PASS"
  //a_info_field := fmt.Sprintf("END=%d", a_start+a_len)
  a_info_field := fmt.Sprintf("END=%d", a_start+a_len-1)

  if info.vartype == pasta.NOC {
    a_filt_field = "NOCALL"
  }

  //                            0   1   2   3   4   5    6  7   8   9
  out.WriteString( fmt.Sprintf("%s\t%d\t%s\t%c\t%s\t%s\t%s\t%s\t%s\t%s\n",
    g.ChromStr,
    a_start,
    g.Id,
    a_ref_bp,
    alt_field,
    g.Qual,
    a_filt_field,
    a_info_field,
    g.Format,
    a_gt_field) )


}

func (g *GVCFRefVar) _emit_ref_left_anchor(info GVCFRefVarInfo, out *bufio.Writer) {
  a_start := info.ref_start+1
  a_len := info.ref_len
  a_r_seq := info.refseq
  a_ref_bp := byte('.')
  if len(a_r_seq)>0 { a_ref_bp = a_r_seq[0] }
  a_gt_field := "0/0"

  a_filt_field := "PASS"
  //a_info_field := fmt.Sprintf("END=%d", a_start+a_len)
  a_info_field := fmt.Sprintf("END=%d", a_start+a_len-1)

  //                            0   1   2   3   4   5    6  7   8   9
  out.WriteString( fmt.Sprintf("%s\t%d\t%s\t%c\t%s\t%s\t%s\t%s\t%s\t%s\n",
    g.ChromStr,
    a_start,
    g.Id,
    a_ref_bp,
    ".",
    g.Qual,
    a_filt_field,
    a_info_field,
    g.Format,
    a_gt_field) )

}


func (g *GVCFRefVar) _emit_alt_left_anchor_p(info GVCFRefVarInfo, z byte, out *bufio.Writer) {

  b_r_seq := info.refseq
  b_refseq,b_alt,b_gt_field := g._ref_alt_gt_fields(b_r_seq, info.altseq)

  _ = b_refseq


  _a := []string{}
  for ii:=0; ii<len(b_alt); ii++ {

    if g.BeginningAltCondition {
      _a = append(_a, fmt.Sprintf("%s", b_alt[ii]))
    } else {
      _a = append(_a, fmt.Sprintf("%c%s", z, b_alt[ii]))
    }
  }
  b_alt_field := strings.Join(_a, ",")

  b_start := info.ref_start
  b_len := info.ref_len+1
  b_ref_bp := z
  b_filt_field := "PASS"
  //b_info_field := fmt.Sprintf("END=%d", b_start+b_len)
  b_info_field := fmt.Sprintf("END=%d", b_start+b_len-1)



  //                            0   1   2   3   4   5    6  7   8   9
  out.WriteString( fmt.Sprintf("%s\t%d\t%s\t%c\t%s\t%s\t%s\t%s\t%s\t%s\n",
    g.ChromStr,
    b_start,
    g.Id,
    b_ref_bp,
    b_alt_field,
    g.Qual,
    b_filt_field,
    b_info_field,
    g.Format,
    b_gt_field) )

}

// (g)VCF lines consist of:
//
// 0      1     2   3   4   5    6      7    8      9
// chrom  pos   id  ref alt qual filter info format sample
//
// Print receives interpreted lines at a time.
//
// We make a simplifying assumption that if there is a nocall region right next to
// an alt call, the alt call gets subsumed into the nocall region.
//
// We print the nocall region with full sequence so that it's recoverable but otherwise it
// looks like a nocall region.
//
func (g *GVCFRefVar) Print(vartype int, ref_start, ref_len int, refseq []byte, altseq [][]byte, out *bufio.Writer) error {
  local_debug := false

  if g.PrintHeader {
    g.Header(out)
    g.PrintHeader = false
  }

  vi := GVCFRefVarInfo{}
  vi.vartype = vartype
  vi.ref_start = ref_start
  vi.ref_len = ref_len
  vi.refseq = string(refseq)
  for ii:=0; ii<len(altseq); ii++ {
    vi.altseq = append(vi.altseq, string(altseq[ii]))
  }
  vi.chrom = g.ChromStr

  g.StateHistory = append(g.StateHistory, vi)

  processing:=false
  if len(g.StateHistory)>1 { processing = true }

  if local_debug {
    fmt.Printf("\n")
    fmt.Printf("vartype: %d (REF %d, NOC %d, ALT %d)\n", vartype, pasta.REF, pasta.NOC, pasta.ALT)
    fmt.Printf("ref_start: %d, ref_len: %d\n", ref_start, ref_len)
    fmt.Printf("refseq: %s\n", refseq)
    fmt.Printf("altseq: %s\n", altseq)
  }

  // There's a special case when we start with an ALT straight away
  // with no REF before it.  In this case we need to take some special
  // consideration not to print the anchor reference base in the anchor
  // sequence as it's a straight substitution.
  //
  if (len(g.StateHistory)==1) && (g.StateHistory[0].vartype == pasta.ALT) {
    g.BeginningAltCondition = true
  }


  for processing && (len(g.StateHistory)>1) {

    if local_debug {
      fmt.Printf("  cp1\n")
    }

    idx:=1

    if g.StateHistory[idx-1].vartype == pasta.REF  {

      if g.StateHistory[idx].vartype==pasta.REF {

        g._emit_ref_left_anchor(g.StateHistory[idx-1], out)
        g.StateHistory = g.StateHistory[idx:]
        continue

      } else if g.StateHistory[idx].vartype==pasta.NOC {

        b_ref,b_alt,_ := g._ref_alt_gt_fields(g.StateHistory[idx].refseq, g.StateHistory[idx].altseq)

        /*
        if len(b_alt)==0 {
          panic(fmt.Sprintf("!!!!!!! b_alt IS ZERO: ref_start %d, ref_len %d, refseq %s, altseq %v, vartype %d, idx %d, state history %v\n",
            ref_start, ref_len, refseq, altseq, vartype, idx, g.StateHistory[idx]))
        }
        */

        // b_alt == 0 -> it's a nocall for both reference and alt
        //
        min_alt_len := 0
        if len(b_alt)>0 {
          min_alt_len = len(b_alt[0])
          for ii:=1; ii<len(b_alt); ii++ {
            if min_alt_len > len(b_alt[ii]) {
              min_alt_len = len(b_alt[ii])
            }
          }
        }

        if min_alt_len>0 {

          // The nocall alt will have a reference anchor so we can
          // emit the current reference
          //
          g._emit_ref_left_anchor(g.StateHistory[idx-1], out)
          g.StateHistory = g.StateHistory[idx:]

        } else {

          n := len(g.StateHistory[idx-1].refseq)
          z := g.StateHistory[idx-1].refseq[n-1]

          g.StateHistory[idx-1].refseq = g.StateHistory[idx-1].refseq[0:n-1]
          g.StateHistory[idx-1].ref_len--
          if g.StateHistory[idx-1].ref_len > 0 {
            g._emit_ref_left_anchor(g.StateHistory[idx-1], out)
          }

          g.StateHistory[idx].refseq = string(z) + b_ref
          g.StateHistory[idx].ref_start--
          g.StateHistory[idx].ref_len++
          for ii:=0; ii<len(g.StateHistory[idx].altseq); ii++ {
            if g.StateHistory[idx].altseq[ii] == "-" {
              g.StateHistory[idx].altseq[ii] = string(z)
            } else {
              g.StateHistory[idx].altseq[ii] = string(z) + g.StateHistory[idx].altseq[ii]
            }
          }

          g.StateHistory = g.StateHistory[idx:]
        }

        continue

      } else if g.StateHistory[idx].vartype==pasta.ALT {

        _,b_alt,_ := g._ref_alt_gt_fields(g.StateHistory[idx].refseq, g.StateHistory[idx].altseq)

        min_alt_len := len(b_alt[0])
        for ii:=1; ii<len(b_alt); ii++ {
          if min_alt_len > len(b_alt[ii]) { min_alt_len = len(b_alt[ii]) }
        }

        if (ref_len>0) && (min_alt_len>0) {

          // Not a straight deletion, we can use a reference base
          // as anchor straight out
          //
          g._emit_ref_left_anchor(g.StateHistory[idx-1], out)
          g._emit_alt_left_anchor(g.StateHistory[idx], out)
          g.StateHistory = g.StateHistory[idx+1:]

        } else {

          // The alt is a straight deletion or insertion
          // so we need to peel off a reference base from
          // the previous line and use it as the anchor base.
          //
          n := len(g.StateHistory[idx-1].refseq)
          z := g.StateHistory[idx-1].refseq[n-1]

          g.StateHistory[idx-1].refseq = g.StateHistory[idx-1].refseq[0:n-1]
          g.StateHistory[idx-1].ref_len--
          if g.StateHistory[idx-1].ref_len > 0 {
            g._emit_ref_left_anchor(g.StateHistory[idx-1], out)
          }
          g._emit_alt_left_anchor_p(g.StateHistory[idx], z, out)

          g.StateHistory = g.StateHistory[idx+1:]
        }

      }

    } else if g.StateHistory[idx-1].vartype == pasta.ALT {

      if g.StateHistory[idx].vartype == pasta.REF {

        _,a_alt,_ := g._ref_alt_gt_fields(g.StateHistory[idx-1].refseq, g.StateHistory[idx-1].altseq)
        prv_min_alt_len := len(a_alt[0])
        for ii:=1; ii<len(a_alt); ii++ {
          if prv_min_alt_len > len(a_alt[ii]) { prv_min_alt_len = len(a_alt[ii]) }
        }
        prv_ref_len := g.StateHistory[idx-1].ref_len


        if (prv_ref_len>0) && (prv_min_alt_len>0) {

          if local_debug { fmt.Printf("  cp1.a\n") }

          // Previous reference length > 0 which means we can use the first
          // base in the reference sequence because there will be at least one
          // substitution.
          //
          g._emit_alt_left_anchor(g.StateHistory[idx-1], out)
          g.StateHistory = g.StateHistory[idx:]

          g.BeginningAltCondition = false
          continue

        } else {

          if local_debug { fmt.Printf("  cp1.b\n") }

          // Else it's a straight deletion (reflen==0), so use
          // a reference base from the end of the sequence
          //

          n := len(g.StateHistory[idx].refseq) ;  _ = n
          z := g.StateHistory[idx].refseq[0] ; _ = z

          g.StateHistory[idx].refseq = g.StateHistory[idx].refseq[1:]
          g.StateHistory[idx].ref_start++
          g.StateHistory[idx].ref_len--

          g._emit_alt_left_anchor_p(g.StateHistory[idx-1], z, out)

          if g.StateHistory[idx].ref_len == 0 {

            // The ref line was only 1 ref base long so
            // discard it
            //
            g.StateHistory = g.StateHistory[idx+1:]

          } else {
            g.StateHistory = g.StateHistory[idx:]
          }

          g.BeginningAltCondition = false
          continue

        }

      } else if g.StateHistory[idx].vartype == pasta.ALT {

        _,a_alt,_ := g._ref_alt_gt_fields(g.StateHistory[idx-1].refseq, g.StateHistory[idx-1].altseq)
        _,b_alt,_ := g._ref_alt_gt_fields(g.StateHistory[idx].refseq, g.StateHistory[idx].altseq)

        // Subsume the previous ALT into the current NOC entry
        //
        g.StateHistory[idx].ref_start = g.StateHistory[idx-1].ref_start
        g.StateHistory[idx].ref_len += g.StateHistory[idx-1].ref_len

        g.StateHistory[idx].altseq = g.StateHistory[idx].altseq[:]
        for ii:=0; ii<len(a_alt); ii++ {
          g.StateHistory[idx].altseq = append(g.StateHistory[idx].altseq, string(a_alt[ii]) + string(b_alt[ii]))
        }

        g.BeginningAltCondition = false
        continue

      } else if g.StateHistory[idx].vartype == pasta.NOC {

        _,a_alt,_ := g._ref_alt_gt_fields(g.StateHistory[idx-1].refseq, g.StateHistory[idx-1].altseq)
        _,b_alt,_ := g._ref_alt_gt_fields(g.StateHistory[idx].refseq, g.StateHistory[idx].altseq)

        // Subsume the previous ALT into the current NOC entry
        //
        g.StateHistory[idx].ref_start = g.StateHistory[idx-1].ref_start
        g.StateHistory[idx].ref_len += g.StateHistory[idx-1].ref_len

        g.StateHistory[idx].altseq = g.StateHistory[idx].altseq[:]
        for ii:=0; ii<len(a_alt); ii++ {
          g.StateHistory[idx].altseq = append(g.StateHistory[idx].altseq, string(a_alt[ii]) + string(b_alt[ii]))
        }

        g.StateHistory = g.StateHistory[idx:]

        g.BeginningAltCondition = false
        continue

      }

    } else if g.StateHistory[idx-1].vartype == pasta.NOC {

      if g.StateHistory[idx].vartype == pasta.REF {

        g._emit_alt_left_anchor(g.StateHistory[idx-1], out)
        g.StateHistory = g.StateHistory[idx:]
        continue

      } else if g.StateHistory[idx].vartype == pasta.ALT {

        a_seqs := []string{}
        b_seqs := []string{}

        for ii:=0; ii<len(g.StateHistory[idx-1].altseq); ii++ {
          if g.StateHistory[idx-1].altseq[ii] == "-" {
            a_seqs = append(a_seqs, "")
          } else {
            a_seqs = append(a_seqs, g.StateHistory[idx-1].altseq[ii])
          }
        }

        for ii:=0; ii<len(g.StateHistory[idx].altseq); ii++ {
          if g.StateHistory[idx].altseq[ii] == "-" {
            b_seqs = append(b_seqs, "")
          } else {
            b_seqs = append(b_seqs, g.StateHistory[idx].altseq[ii])
          }
        }

        // Subsume the previous ALT into the current NOC entry
        //
        g.StateHistory[idx].ref_start = g.StateHistory[idx-1].ref_start
        g.StateHistory[idx].ref_len += g.StateHistory[idx-1].ref_len
        g.StateHistory[idx].vartype = g.StateHistory[idx-1].vartype

        ref_b_pos := 0
        ref_b := make([]byte, len(g.StateHistory[idx-1].refseq) + len(g.StateHistory[idx].refseq))
        for ii:=0; ii<len(g.StateHistory[idx-1].refseq); ii++ {
          if (g.StateHistory[idx-1].refseq[ii] != '-') {
            ref_b[ref_b_pos] = g.StateHistory[idx-1].refseq[ii]
            ref_b_pos++
          }
        }

        for ii:=0; ii<len(g.StateHistory[idx].refseq); ii++ {
          if (g.StateHistory[idx].refseq[ii] != '-') {
            ref_b[ref_b_pos] = g.StateHistory[idx].refseq[ii]
            ref_b_pos++
          }
        }
        g.StateHistory[idx].refseq = string(ref_b[:ref_b_pos])


        g.StateHistory[idx].altseq = []string{}
        for ii:=0; ii<len(a_seqs); ii++ {
          g.StateHistory[idx].altseq = append(g.StateHistory[idx].altseq, a_seqs[ii] + b_seqs[ii])
        }

        g.StateHistory = g.StateHistory[idx:]
        continue

      } else if g.StateHistory[idx].vartype == pasta.NOC {

        _,a_alt,_ := g._ref_alt_gt_fields(g.StateHistory[idx-1].refseq, g.StateHistory[idx-1].altseq)

        if len(a_alt)==0 {
          g._emit_alt_left_anchor(g.StateHistory[idx-1], out)
          g.StateHistory = g.StateHistory[idx:]
          continue
        }

        min_length:=len(a_alt[0])
        for ii:=1; ii<len(a_alt); ii++ {
          if min_length < len(a_alt[ii]) { min_length = len(a_alt[ii]) }
        }

        if min_length == 0 {
          g.StateHistory = g.StateHistory[idx:]
          continue
        }

        g._emit_alt_left_anchor(g.StateHistory[idx-1], out)
        g.StateHistory = g.StateHistory[idx:]
        continue

      }

    }

    if len(g.StateHistory) < 2 { processing = false }
  }

  return nil
}

func (g *GVCFRefVar) Print_old(vartype int, ref_start, ref_len int, refseq []byte, altseq [][]byte, out *bufio.Writer) error {

  if g.PrintHeader {
    g.Header(out)
    g.PrintHeader = false
  }

  vi := GVCFRefVarInfo{}
  vi.vartype = vartype
  vi.ref_start = ref_start
  vi.ref_len = ref_len
  vi.refseq = string(refseq)
  for ii:=0; ii<len(altseq); ii++ {
    vi.altseq = append(vi.altseq, string(altseq[ii]))
  }
  vi.chrom = g.ChromStr

  g.StateHistory = append(g.StateHistory, vi)


  processing:=false
  if len(g.StateHistory)>1 { processing = true }
  for processing {
    idx:=1

    if g.StateHistory[idx-1].vartype == pasta.REF  {

      if g.StateHistory[idx].vartype == pasta.REF {
        g._emit_ref_left_anchor(g.StateHistory[idx-1], out)
        g.StateHistory = g.StateHistory[1:]
        break
      } else if g.StateHistory[idx].vartype == pasta.NOC {
      } else if g.StateHistory[idx].vartype == pasta.ALT {
      }

    } else if g.StateHistory[idx-1].vartype == pasta.REF && ((g.StateHistory[idx].vartype==pasta.ALT) || (g.StateHistory[idx].vartype==pasta.NOC)) {

      _,b_alt,_ := g._ref_alt_gt_fields(g.StateHistory[idx].refseq, g.StateHistory[idx].altseq)

      min_alt_len := len(b_alt[0])
      for ii:=1; ii<len(b_alt); ii++ {
        if min_alt_len > len(b_alt[ii]) { min_alt_len = len(b_alt[ii]) }
      }

      if (ref_len>0) && (min_alt_len>0) {

        // Not a straight deletion, we can use a reference base
        // as anchor straight out
        //
        g._emit_ref_left_anchor(g.StateHistory[idx-1], out)
        g._emit_alt_left_anchor(g.StateHistory[idx], out)
        g.StateHistory = g.StateHistory[idx+1:]

      } else {

        // The alt is a straight deletion or insertion
        // so we need to peel off a reference base from
        // the previous line and use it as the anchor base.
        //
        n := len(g.StateHistory[idx-1].refseq)
        z := g.StateHistory[idx-1].refseq[n-1]

        g.StateHistory[idx-1].refseq = g.StateHistory[idx-1].refseq[0:n-1]
        g.StateHistory[idx-1].ref_len--
        if g.StateHistory[idx-1].ref_len > 0 {
          g._emit_ref_left_anchor(g.StateHistory[idx-1], out)
        }

        g._emit_alt_left_anchor_p(g.StateHistory[idx], z, out)

        g.StateHistory = g.StateHistory[idx+1:]
      }

      if len(g.StateHistory) < 2 { processing = false }

    } else if ((g.StateHistory[idx-1].vartype==pasta.ALT) || (g.StateHistory[idx-1].vartype==pasta.NOC)) && g.StateHistory[idx].vartype == pasta.REF {

      _,a_alt,_ := g._ref_alt_gt_fields(g.StateHistory[idx-1].refseq, g.StateHistory[idx-1].altseq)
      prv_min_alt_len := len(a_alt[0])
      for ii:=1; ii<len(a_alt); ii++ {
        if prv_min_alt_len > len(a_alt[ii]) { prv_min_alt_len = len(a_alt[ii]) }
      }
      prv_ref_len := g.StateHistory[idx-1].ref_len

      if (prv_ref_len>0) && (prv_min_alt_len>0) {

        // Previous reference length > 0 which means we can use the first
        // base in the reference sequence because there will be at least one
        // substitution.
        //
        g._emit_alt_left_anchor(g.StateHistory[idx-1], out)

        g.StateHistory = g.StateHistory[idx:]

      } else {

        // Else it's a straight deletion (reflen==0), so use
        // a reference base from the end of the sequence
        //

        n := len(g.StateHistory[idx].refseq) ;  _ = n
        z := g.StateHistory[idx].refseq[0] ; _ = z

        g.StateHistory[idx].refseq = g.StateHistory[idx].refseq[1:]
        g.StateHistory[idx].ref_start++
        g.StateHistory[idx].ref_len--

        g._emit_alt_left_anchor_p(g.StateHistory[idx-1], z, out)

        if g.StateHistory[idx].ref_len == 0 {

          // The ref line was only 1 ref base long so
          // discard it
          //
          g.StateHistory = g.StateHistory[idx+1:]

        } else {
          g.StateHistory = g.StateHistory[idx:]
        }

      }

    } else if (g.StateHistory[idx-1].vartype==pasta.REF) && (g.StateHistory[idx].vartype==pasta.REF) {

      g.StateHistory[idx].refseq = g.StateHistory[idx-1].refseq + g.StateHistory[idx].refseq
      g.StateHistory[idx].ref_start = g.StateHistory[idx-1].ref_start
      g.StateHistory[idx].ref_len += g.StateHistory[idx-1].ref_len
      g.StateHistory = g.StateHistory[idx:]

    } else if g.StateHistory[idx-1].vartype == pasta.REF {
      g.StateHistory = g.StateHistory[idx+1:]
    } else  {
      fmt.Printf(">>>>\n%v\n", g.StateHistory)
      panic("inalid option")
    }

    if len(g.StateHistory) < 2 { processing = false }
  }


  return nil
}

func process_ref_alt_seq(refseq []byte, altseq [][]byte) (string,bool) {
  var type_str string
  noc_flag := false
  indel_flag := false
  n1 := []byte{'n'}

  len_match := true
  for ii:=0; ii<len(altseq); ii++ {
    if len(altseq[ii])!=len(refseq) {
      len_match = false
      break
    }
  }

  if (len(refseq)==1) && len_match {
    for ii:=0; ii<len(altseq); ii++ {
      if altseq[ii][0]=='-' { indel_flag = true; break }
    }
  }

  if len_match && (len(refseq)==1) {
    if indel_flag || (refseq[0]=='-') {
      type_str = "INDEL"
    } else {
      type_str = "SNP"
    }
  } else if len_match {
    type_str = "SUB"

    // In the case:
    // * it's a non 0-length string
    // * the lengths of the altseqs match the refseq
    // * the altseqs are all 'n' (nocall)
    // -> it's a 'true' nocall line
    //
    if len(refseq)>0 {
      noc_flag = true
      for a:=0; a<len(altseq); a++ {
        n := bytes.Count(altseq[a], n1)
        if n!=len(altseq[a]) {
          noc_flag = false
          break
        }
      }
      if noc_flag { type_str = "NOC" }
    }
  } else {
    type_str = "INDEL"
  }

  return type_str, noc_flag
}


func (g *GVCFRefVar) PrintEnd(out *bufio.Writer) error {

  idx:=0

  if g.StateHistory[idx].vartype==pasta.REF {
    g._emit_ref_left_anchor(g.StateHistory[idx], out)
  } else if g.StateHistory[idx].vartype==pasta.NOC {
    g._emit_alt_left_anchor(g.StateHistory[idx], out)
  } else if g.StateHistory[idx].vartype==pasta.ALT {
    g._emit_alt_left_anchor(g.StateHistory[idx], out)
  }

  out.Flush()

  return nil
}

//---

func (g *GVCFRefVar) PastaBegin(out *bufio.Writer) error {
  return nil
}

func (g *GVCFRefVar) PastaEnd(out *bufio.Writer) error {

  out.Flush()
  return nil
}

func (g *GVCFRefVar) _parse_info_field_value(info_line string, field string, sep string) (string, error) {
  sa := strings.Split(info_line, sep)
  for ii:=0; ii<len(sa); ii++ {
    fv := strings.Split(sa[ii], "=")
    if len(fv)!=2 { return "", fmt.Errorf("invalud field") }

    if fv[0] == field { return fv[1], nil }
  }
  return "", fmt.Errorf("field not found")
}

func (g *GVCFRefVar) _parameter_index(line string, field string, sep string) (int, error) {
  sa := strings.Split(line, sep)
  for ii:=0; ii<len(sa); ii++ {
    if sa[ii] == field { return ii, nil }
  }
  return -1, fmt.Errorf("field not found")
}

func (g *GVCFRefVar) _get_gt_array(gt_str string, ploidy int) ([]int, error) {
  gt_array := []int{}
  if !strings.ContainsAny(gt_str, "|/") {
    v,e := strconv.Atoi(gt_str)
    if e!=nil { return nil, e }
    gt_array = append(gt_array, v)
    return gt_array, nil
  }

  _sa := strings.Split(gt_str, "/")
  if len(_sa)==1 {
    _sa = strings.Split(gt_str, "|")
  }

  if len(_sa)>ploidy { return nil, fmt.Errorf("invalid GT field") }

  for ii:=0; ii<ploidy; ii++ {
    if ii < len(_sa) {
      v,e := strconv.Atoi(_sa[ii])
      if e!=nil { return nil, e }
      gt_array = append(gt_array, v)
    } else {
      gt_array = append(gt_array, gt_array[ii-1])
    }
  }

  return gt_array, nil
}

func (g *GVCFRefVar) Pasta(gvcf_line string, ref_stream *bufio.Reader, out *bufio.Writer) error {
  var err error
  CHROM_FIELD_POS := 0 ; _ = CHROM_FIELD_POS
  START_FIELD_POS := 1 ; _ = START_FIELD_POS
  ID_FIELD_POS := 2 ; _ = ID_FIELD_POS
  REF_FIELD_POS := 3 ; _ = REF_FIELD_POS
  ALT_FIELD_POS := 4 ; _ = ALT_FIELD_POS
  QUAL_FIELD_POS := 5 ; _ = QUAL_FIELD_POS
  FILTER_FIELD_POS := 6 ; _ = FILTER_FIELD_POS
  INFO_FIELD_POS := 7 ; _ = INFO_FIELD_POS
  FORMAT_FIELD_POS := 8 ; _ = FORMAT_FIELD_POS
  SAMPLE0_FIELD_POS := 9 ; _ = SAMPLE0_FIELD_POS

  // empty line or comment
  //
  if (len(gvcf_line)==0) || (gvcf_line[0]=='#') { return nil }


  line_part := strings.Split(gvcf_line, "\t")

  _start,e := strconv.Atoi(line_part[START_FIELD_POS])
  if e!=nil { return e }

  _end_str,e := g._parse_info_field_value(line_part[INFO_FIELD_POS], "END", ":")
  _end := -1
  if e==nil {
    _end,err = strconv.Atoi(_end_str)
    if err!=nil { return err }
  }
  if _end==-1 { _end = _start+1 }

  alt_seq := []string{}
  if line_part[ALT_FIELD_POS]!="." {
    alt_seq = strings.Split(line_part[ALT_FIELD_POS], ",")
  }

  gt_samp_idx,e := g._parameter_index(line_part[FORMAT_FIELD_POS], "GT", ":")
  if e!=nil { return e }

  samp_part := strings.Split(line_part[SAMPLE0_FIELD_POS], ":")
  if gt_samp_idx >= len(samp_part) { return fmt.Errorf("GT index overflow") }

  n_allele := 2
  samp_str := samp_part[gt_samp_idx]
  samp_seq_idx,e := g._get_gt_array(samp_str, n_allele)
  if e!=nil { return e }

  ref_anchor_base := line_part[REF_FIELD_POS]
  //refn := _end - _start
  refn := (_end + 1) - _start

  if (samp_seq_idx[0] == samp_seq_idx[1]) && (samp_seq_idx[0] == 0) {

    for ii:=0; ii<refn; ii++ {
      stream_ref_bp,e := ref_stream.ReadByte()
      if e!=nil { return e }
      for stream_ref_bp == '\n' || stream_ref_bp == ' ' || stream_ref_bp == '\t' || stream_ref_bp == '\r' {
        stream_ref_bp,e = ref_stream.ReadByte()
        if e!=nil { return e }
      }


      for a:=0; a<n_allele; a++ {

        if (g.LFMod>0) && (g.OCounter > 0) && ((g.OCounter%g.LFMod)==0) {
          out.WriteByte('\n')
        }
        g.OCounter++

        out.WriteByte(stream_ref_bp)
      }

    }

    return nil
  }

  mM := refn
  for ii:=0; ii<n_allele; ii++ {

    // reference
    //
    if samp_seq_idx[ii]==0 { continue }

    // find maximum of alt sequence lengths
    //
    a_idx := samp_seq_idx[ii]-1
    if mM < len(alt_seq[a_idx]) { mM = len(alt_seq[a_idx]) }
  }



  // Loop through, emitting the appropriate substitution
  // if we have a reference, a deletion if the alt sequence
  // has run out or an insertion if the reference sequence has
  // run out.
  //
  // The reference is 'shifted' to the left, which means there
  // will be (potentially 0-length) substitutions followed by
  // (potentially 0-length) insertions and/or deletions.
  //
  for i:=0; i<mM; i++  {

    // Get the reference base
    //
    var stream_ref_bp byte
    if i<refn {

      stream_ref_bp,e = ref_stream.ReadByte()
      if e!=nil { return e }
      for stream_ref_bp == '\n' || stream_ref_bp == ' ' || stream_ref_bp == '\t' || stream_ref_bp == '\r' {
        stream_ref_bp,e = ref_stream.ReadByte()
        if e!=nil { return e }
      }

    }


    if (refn>0) && (i==0) && (stream_ref_bp!=ref_anchor_base[0]) {
      return fmt.Errorf(fmt.Sprintf("stream reference (%c) does not match VCF ref base (%c) at position %d\n", stream_ref_bp, ref_anchor_base[0], _start))
    }
    _ = stream_ref_bp

    // Emit a symbol per alt sequence
    //
    for a:=0; a<n_allele; a++ {

      var bp_ref byte = '-'
      if i<refn {
        bp_ref = stream_ref_bp
        if bp_ref != stream_ref_bp {
          return fmt.Errorf( fmt.Sprintf("ref stream to gff ref mismatch (ref stream %c != gff ref %c @ %d)", stream_ref_bp, bp_ref, g.RefPos) )
        }
      }

      var bp_alt byte = '-'
      if samp_seq_idx[a]==0 {
        bp_alt = bp_ref
      } else {
        a_idx := samp_seq_idx[a]-1
        if i<len(alt_seq[a_idx]) { bp_alt = alt_seq[a_idx][i] }
      }

      pasta_ch := pasta.SubMap[bp_ref][bp_alt]
      if pasta_ch == 0 { return fmt.Errorf("invalid character") }

      if (g.LFMod>0) && (g.OCounter > 0) && ((g.OCounter%g.LFMod)==0) {
        out.WriteByte('\n')
      }
      g.OCounter++


      out.WriteByte(pasta_ch)
    }

  }

  return nil
}
