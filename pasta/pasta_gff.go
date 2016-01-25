package main

import "fmt"
import "strconv"
import "strings"
import "bufio"
import "os"
import "io"

import "time"

import "github.com/abeconnelly/pasta"
import "github.com/abeconnelly/simplestream"

type GFFRefVar struct {
  Type int
  MessageType int
  RefSeqFlag bool
  NocSeqFlag bool
  Out io.Writer
  Msg ControlMessage
  RefBP byte
  Allele int

  ChromStr string
  SrcStr string
  RefPos int

  PrintHeader bool
  //Header string
  Reference string
}

func (g *GFFRefVar) Init() {
  g.PrintHeader = true
  g.Reference = "unk"

  g.ChromStr = "Unk"
  g.SrcStr = "unk"
  g.RefPos = 0
  g.Allele = 2

}

func (g *GFFRefVar) Chrom(chr string) { g.ChromStr = chr }
func (g *GFFRefVar) Pos(pos int) { g.RefPos = pos }

func (g *GFFRefVar) Header() string {
//func gff_header(info *GFFVarInfo) string {

  header := []string{}

  t := time.Now()
  str_time := fmt.Sprintf("%v", t.Format(time.RFC3339))

  header = append(header, fmt.Sprintf("## genome-build %s", g.Reference))
  header = append(header, fmt.Sprintf("# File creation date: %s", str_time))
  header = append(header, "#>chrom\tsource\tvartype\tbegin\tend\t.\t+\t.\tseq")

  return strings.Join(header, "\n") + "\n"
}


func (g *GFFRefVar) Print(vartype int, ref_start, ref_len int, refseq []byte, altseq [][]byte) error {
//func gff_printer(vartype int, ref_start, ref_len int, refseq []byte, altseq [][]byte, info_if interface{}) error {

  //info := info_if.(*GFFVarInfo) ; _ = info
  out := os.Stdout

  if g.PrintHeader {
    g.PrintHeader = false
    h := g.Header()
    out.WriteString(h)

  }


  chrom := g.ChromStr
  src := g.SrcStr
  type_str := "REF"
  seq_str := "."
  if vartype == NOC {
    type_str = "NOC"
  } else if vartype == ALT {

    len_match := true
    for ii:=0; ii<len(altseq); ii++ {
      if len(altseq[ii])!=len(refseq) {
        len_match = false
        break
      }
    }

    if len_match && (len(refseq)==1) {
      type_str = "SNP"
    } else if len_match {
      type_str = "SUB"
    } else {
      type_str = "INDEL"
    }

    alt_a := []string{}
    for ii:=0; ii<len(altseq); ii++ {
      if len(altseq[ii])==0 {
        alt_a = append(alt_a, "-")
      } else {
        alt_a = append(alt_a, string(altseq[ii]))
      }
    }

    r_s := "-"
    if len(refseq) > 0 { r_s = string(refseq) }

    seq_str = fmt.Sprintf("alleles %s;ref_allele %s", strings.Join(alt_a, "/"), r_s)
  }

  // GFF is 1-base (starts at 1, not 0), end inclusive
  //

  if vartype == REF {
    out.WriteString( fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t.\t+\t.\t%s\n", chrom, src, type_str, ref_start+1, ref_start+ref_len, seq_str) )
  } else if vartype == NOC {
    out.WriteString( fmt.Sprintf("#%s\t%s\t%s\t%d\t%d\t.\t+\t.\t%s\n", chrom, src, type_str, ref_start+1, ref_start+ref_len, seq_str) )
  } else if vartype == ALT {
    out.WriteString( fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t.\t+\t.\t%s\n", chrom, src, type_str, ref_start+1, ref_start+ref_len, seq_str) )
  }


  return nil
}

func _tol(A string) string {
  z := make([]byte, len(A))
  for i:=0; i<len(A); i++ {
    if A[i] >= 'A' && A[i] <= 'Z' {
      z[i] = A[i] - 'A' + 'a'
    } else {
      z[i] = A[i]
    }
  }
  return string(z)
}

func (g *GFFRefVar) _gff_parse_refstr(seq_str string) (string, error) {
  parts := strings.Split(seq_str, ";")

  for i:=0; i<len(parts); i++ {
    if strings.HasPrefix(parts[i], "ref_allele ") {

      _x := strings.Split(parts[i], " ")
      if len(_x)!=2 {
        return "", fmt.Errorf("no ref sequence found")
      }

      if _x[1] == "-" { return "", nil }
      return _tol(_x[1]), nil
    }
  }

  return "", fmt.Errorf("no 'ref_allele' found")
}

func (g *GFFRefVar) _gff_parse_allele(seq_str string) (_z []string, e error) {
  parts := strings.Split(seq_str, ";")

  for i:=0; i<len(parts); i++ {
    if strings.HasPrefix(parts[i], "alleles ") {

      _x := strings.Split(parts[i], " ")
      if len(_x)!=2 {
        e = fmt.Errorf("no alternate alleles found")
        return
      }

      if (strings.Index(_x[1], "/")>=0) && (strings.Index(_x[1],"|")>=0) {
        e = fmt.Errorf("cannot interpret alternate alleles (has both '/' and '|')")
        return
      }

      if strings.Index(_x[1], "/")>=0 {
        _y := strings.Split(_x[1], "/")
        for ii:=0; ii<len(_y); ii++ {
          if _y[ii] == "-" {
            _z = append(_z, "")
          } else {
            _z = append(_z, _tol(_y[ii]))
          }
        }
        return
      }

      if strings.Index(_x[1], "|")>=0 {
        _y := strings.Split(_x[1], "/")
        for ii:=0; ii<len(_y); ii++ {
          if _y[ii] == "-" {
            _z = append(_z, "")
          } else {
            _z = append(_z, _tol(_y[ii]))
          }
        }
        return
      }

      _z = append(_z, _x[1])
      for a:=1; a<g.Allele; a++ {
        if _x[1] == "-" {
          _z = append(_z, "")
        } else {
          _z = append(_z, _tol(_x[1]))
        }
      }

      return
    }
  }

  e = fmt.Errorf("no 'alleles' found")
  return
}

//func (g *GFFRefVar) Pasta(gff_line string, ref_stream *simplestream.SimpleStream, out io.Writer) error {
func (g *GFFRefVar) Pasta(gff_line string, ref_stream *simplestream.SimpleStream, out *bufio.Writer) error {

  if len(gff_line)==0 { return nil }
  if gff_line[0] == '\n' { return nil }
  if gff_line[0] == '#' { return nil }
  if gff_line[0] == '>' { return nil }

  line_parts := strings.Split(gff_line, "\t")
  chrom := line_parts[0] ; _ = chrom
  src := line_parts[1] ; _ = src
  vartype := line_parts[2] ; _ = vartype
  beg_s_1ref := line_parts[3] ; _ = beg_s_1ref
  end_s_1ref := line_parts[4] ; _ = end_s_1ref
  x := line_parts[5] ; _ = x
  y := line_parts[6] ; _ = y
  z := line_parts[7] ; _ = z
  seq_str := line_parts[8] ; _ = seq_str

  beg64,e := strconv.ParseInt(beg_s_1ref, 10, 64)
  if e!=nil { return e }

  end64,e := strconv.ParseInt(end_s_1ref, 10, 64)
  if e!=nil { return e }

  n := end64-beg64+1

  if vartype == "REF" {
    for i:=int64(0); i<n; i++ {
      b,e := ref_stream.Getc()
      if e!=nil { return e }
      for b == '\n' || b == ' ' || b == '\t' || b == '\r' {
        b,e = ref_stream.Getc()
        if e!=nil { return e }
      }
      out.WriteByte(b)

      for a:=1; a<g.Allele; a++ { out.WriteByte(b) }
      g.RefPos++
    }
    return nil
  }

  allele_str,e := g._gff_parse_allele(seq_str)
  if e!=nil { return e }

  ref_str,e := g._gff_parse_refstr(seq_str)
  if e!=nil { return e }

  if int64(len(ref_str)) != n {
    return fmt.Errorf( fmt.Sprintf("ref sequence length mismatch (len(%s) = %d) != (%d - %d + 1 = %d)",
      ref_str, len(ref_str), end64, beg64, n) )
  }

  mM := len(ref_str)
  for i:=0; i<len(allele_str); i++ {
    if mM < len(allele_str[i]) { mM = len(allele_str[i]) }
  }

  for i:=0; i<mM; i++  {

    var stream_ref_bp byte
    if i<len(ref_str) {
      stream_ref_bp,e = ref_stream.Getc()
      if e!=nil { return e }
      for stream_ref_bp == '\n' || stream_ref_bp == ' ' || stream_ref_bp == '\t' || stream_ref_bp == '\r' {
        stream_ref_bp,e = ref_stream.Getc()
        if e!=nil { return e }
      }
    }
    _ = stream_ref_bp

    for a:=0; a<len(allele_str); a++ {

      var bp_ref byte = '-'
      //bp_ref := "-"
      if i<len(ref_str) {
        bp_ref = ref_str[i]
        if bp_ref != stream_ref_bp {
          return fmt.Errorf( fmt.Sprintf("ref stream to gff ref mismatch (ref stream %c != gff ref %c @ %d)", stream_ref_bp, bp_ref, g.RefPos) )
        }
      }

      var bp_alt byte = '-'
      //bp_alt := "-"
      if i<len(allele_str[a]) { bp_alt = allele_str[a][i] }


      pasta_ch := pasta.SubMap[bp_ref][bp_alt]
      if pasta_ch == 0 {
        return fmt.Errorf("invalid character")
      }

      out.WriteByte(pasta_ch)

    }
  }

  //out.WriteByte('\n')
  out.Flush()

  return nil
}

