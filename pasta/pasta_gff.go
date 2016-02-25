package main

import "fmt"
import "strconv"
import "strings"
import "bufio"
import "io"

import "bytes"

import "time"

import "github.com/abeconnelly/pasta"

type GFFRefVar struct {
  Type int
  MessageType int
  RefSeqFlag bool
  NocSeqFlag bool
  Out io.Writer
  Msg ControlMessage
  RefBP byte
  Allele int

  ShowNoCallFlag bool

  ChromStr string
  SrcStr string
  RefPos int

  PrevChromStr string
  PrevRefPos int

  OCounter int
  LFMod int

  PrintHeader bool
  ChromUpdate bool
  RefPosUpdate bool
  Reference string
}

func (g *GFFRefVar) Init() {
  g.PrintHeader = true
  g.Reference = "unk"

  g.ChromStr = "Unk"
  g.SrcStr = "."
  g.RefPos = 0
  g.Allele = 2

  g.PrevChromStr = "Unk"
  g.PrevRefPos = 0

  g.OCounter = 0
  g.LFMod = 50

  g.ShowNoCallFlag = false
  //g.ShowNoCallFlag = true
  g.ChromUpdate = false
  g.RefPosUpdate = false

}

func (g *GFFRefVar) Chrom(chr string) {
  g.ChromStr = chr
  g.ChromUpdate = true
}

func (g *GFFRefVar) Pos(pos int) {
  g.RefPos = pos
  g.PrevRefPos = pos
  g.RefPosUpdate = true
}

func (g *GFFRefVar) Header(out *bufio.Writer) error {

  header := []string{}

  t := time.Now()
  str_time := fmt.Sprintf("%v", t.Format(time.RFC3339))

  header = append(header, fmt.Sprintf("## genome-build %s", g.Reference))
  header = append(header, fmt.Sprintf("# File creation date: %s", str_time))
  header = append(header, "#>chrom\tsource\tvartype\tbegin\tend\t.\t+\t.\tseq")

  out.WriteString( strings.Join(header, "\n") + "\n" )

  return nil
}


func (g *GFFRefVar) Print(vartype int, ref_start, ref_len int, refseq []byte, altseq [][]byte, out *bufio.Writer) error {

  if g.PrintHeader {
    g.PrintHeader = false
    e := g.Header(out)
    if e!=nil { return e}
  }

  indel_flag := false

  n1 := []byte{'n'}
  chrom := g.ChromStr
  src := g.SrcStr
  type_str := "REF"
  seq_str := "."

  if vartype == NOC || vartype == ALT {

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
        noc_flag := true
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

    if type_str == "NOC" {
      if g.ShowNoCallFlag {
        out.WriteString( fmt.Sprintf("#%s\t%s\t%s\t%d\t%d\t.\t+\t.\t%s\n", chrom, src, type_str, ref_start+1, ref_start+ref_len, seq_str) )
      }
    } else {
      out.WriteString( fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t.\t+\t.\t%s\n", chrom, src, type_str, ref_start+1, ref_start+ref_len, seq_str) )
    }

  } else if vartype == ALT {
    out.WriteString( fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t.\t+\t.\t%s\n", chrom, src, type_str, ref_start+1, ref_start+ref_len, seq_str) )
  }


  return nil
}

// to lower [a-z]
//
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

// Parse the 'ref_allele' portion in the 'sequence' field. e.g.:
//
// ... allele acat/tcat;ref_allele gcat
//
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

// Parse the 'allele' portion in the 'sequence field .e.g:
//
// ... allele acat/tcat;ref_allele gcat
// ... allele ctag;ref_allele gcat
//
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

func (g *GFFRefVar) PrintEnd(out *bufio.Writer) error {
  out.Flush()
  return nil
}

// Header for PASTA stream
//
func (g *GFFRefVar) PastaBegin(out *bufio.Writer) error {
  out.WriteString( fmt.Sprintf(">C{%s}>P{%d}\n", g.ChromStr, g.RefPos) )
  return nil
}

// Footer for PASTA stream
//
func (g *GFFRefVar) PastaEnd(out *bufio.Writer) error {
  out.WriteByte('\n')
  out.Flush()
  return nil
}

// Called on each GFF line evaluation
//
func (g *GFFRefVar) Pasta(gff_line string, ref_stream *bufio.Reader, out *bufio.Writer) error {

  if len(gff_line)==0 { return nil }
  if gff_line[0] == '\n' { return nil }
  if gff_line[0] == '#' { return nil }
  if gff_line[0] == '>' { return nil }

  // Print header if there are any new updates
  //
  if g.ChromUpdate {
    out.WriteString( fmt.Sprintf(">C{%s}", g.ChromStr) )
  }

  if g.RefPosUpdate {
    out.WriteString( fmt.Sprintf(">P{%d}", g.RefPos) )
  }

  if g.ChromUpdate || g.RefPosUpdate {
    out.WriteByte('\n')
  }

  g.ChromUpdate = false
  g.RefPosUpdate = false


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

  beg64_0ref,e := strconv.ParseInt(beg_s_1ref, 10, 64)
  if e!=nil { return e }
  beg64_0ref--

  end64_0ref,e := strconv.ParseInt(end_s_1ref, 10, 64)
  if e!=nil { return e }
  end64_0ref--

  n := end64_0ref-beg64_0ref+1

  // If we've skipped to a new position, insert
  // the appropriate amount of 'nocalls'.
  //
  if int(beg64_0ref) != g.PrevRefPos {
    dn := int(beg64_0ref) - g.PrevRefPos
    for i:=0; i<dn; i++ {
      b,e := ref_stream.ReadByte()
      if e!=nil { return e }
      for b == '\n' || b == ' ' || b == '\t' || b == '\r' {
        b,e = ref_stream.ReadByte()
        if e!=nil { return e }
      }
      pasta_ch := pasta.SubMap[b]['n']

      for a:=0; a<g.Allele; a++ {

        if (g.LFMod>0) && (g.OCounter > 0) && ((g.OCounter%g.LFMod)==0) {
          out.WriteByte('\n')
        }
        g.OCounter++

        out.WriteByte(pasta_ch)
      }

    }
  }

  // Update position and chromosome state
  //
  g.PrevRefPos = int(beg64_0ref+n)
  g.PrevChromStr = chrom


  // If it's a ref line, peel off ref bases
  // from the reference stream and return.
  //
  if vartype == "REF" {

    for i:=int64(0); i<n; i++ {

      b,e := ref_stream.ReadByte()
      if e!=nil { return e }
      for b == '\n' || b == ' ' || b == '\t' || b == '\r' {
        b,e = ref_stream.ReadByte()
        if e!=nil { return e }
      }

      if (g.LFMod>0) && (g.OCounter > 0) && ((g.OCounter%g.LFMod)==0) {
        out.WriteByte('\n')
      }
      g.OCounter++

      out.WriteByte(b)

      for a:=1; a<g.Allele; a++ {

        if (g.LFMod>0) && (g.OCounter > 0) && ((g.OCounter%g.LFMod)==0) {
          out.WriteByte('\n')
        }
        g.OCounter++

        out.WriteByte(b)
      }
      g.RefPos++
    }

    out.Flush()

    return nil
  }

  // allele_str filled with appropriate allele count
  // copies of string for easy processing below.
  //
  allele_str,e := g._gff_parse_allele(seq_str)
  if e!=nil { return e }

  ref_str,e := g._gff_parse_refstr(seq_str)
  if e!=nil { return e }

  if int64(len(ref_str)) != n {
    return fmt.Errorf( fmt.Sprintf("ref sequence length mismatch (len(%s) = %d) != (%d - %d + 1 = %d)",
      ref_str, len(ref_str), end64_0ref, beg64_0ref, n) )
  }

  // Find the maximum length of the reference and alt sequences
  //
  mM := len(ref_str)
  for i:=0; i<len(allele_str); i++ {
    if mM < len(allele_str[i]) { mM = len(allele_str[i]) }
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
    if (len(ref_str)>0) && (i<len(ref_str)) && (ref_str[0]!='-') {

      stream_ref_bp,e = ref_stream.ReadByte()
      if e!=nil { return e }
      for stream_ref_bp == '\n' || stream_ref_bp == ' ' || stream_ref_bp == '\t' || stream_ref_bp == '\r' {
        stream_ref_bp,e = ref_stream.ReadByte()
        if e!=nil { return e }
      }
    }
    _ = stream_ref_bp

    // Emit a symbol per alt sequence
    //
    for a:=0; a<len(allele_str); a++ {

      var bp_ref byte = '-'
      if i<len(ref_str) {
        bp_ref = ref_str[i]
        if bp_ref != stream_ref_bp {
          return fmt.Errorf( fmt.Sprintf("ref stream to gff ref mismatch (ref stream %c != gff ref %c @ %d)", stream_ref_bp, bp_ref, g.RefPos) )
        }
      }

      var bp_alt byte = '-'
      if i<len(allele_str[a]) { bp_alt = allele_str[a][i] }

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



