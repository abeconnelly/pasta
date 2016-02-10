package main

import "fmt"
import "strconv"
import "strings"
import "bufio"
//import "io"
//import "bytes"

import "time"

import "github.com/abeconnelly/pasta"

const PASTA_CGIVAR_SOFT_VER = "0.1.0"
const PASTA_CGIVAR_FMT_VER_STR = "2.5"

type CGIRefVar struct {
  Locus int
  Ploidy int
  ChromStr string

  Fields []string

  PrintHeader bool

  Date time.Time


  Seq [][]byte
  Start int

  LFMod int
  OCounter int
}

func (g *CGIRefVar) Init() {
  g.Locus = 1
  g.ChromStr = "Unk"
  g.Ploidy=2

  g.Fields = []string{ "locus", "ploidy", "allele",
    "chromosome", "begin", "end", "varType", "reference",
    "alleleSeq", "varScoreVAF", "varScoreEAF", "varFilter",
    "hapLink", "xRef", "alleleFreq", "alternativeCalls" }

  g.PrintHeader = true
  g.Date = time.Now()
  g.Start = 0
  g.Seq = make([][]byte, g.Ploidy)
  for ii:=0; ii<g.Ploidy; ii++ {
    g.Seq[ii] = make([]byte, 0, 1024)
  }

  g.OCounter = 0
  g.LFMod = 50
}

func (g *CGIRefVar) Chrom(chr string) {
}

func (g *CGIRefVar) Pos(pos int) {
}

func (g *CGIRefVar) Header(out *bufio.Writer) error {
  var header = []string{}

  header = append(header, "#GENERATED_BY\tpasta tools")
  header = append(header, fmt.Sprintf("#GENERATED_AT\t%d%02d%02d", g.Date.Year(), g.Date.Month(), g.Date.Day()))
  header = append(header, fmt.Sprintf("#SOFTWARE_VERSION\t%s", PASTA_CGIVAR_SOFT_VER))
  header = append(header, fmt.Sprintf("#FORMAT_VERSION\t%s", PASTA_CGIVAR_FMT_VER_STR))
  header = append(header, "#TYPE\tVAR-ANNOTATION")
  header = append(header, "")

  out.WriteString( strings.Join(header, "\n") )
  out.WriteString("\n")
  out.WriteString( ">" + strings.Join(g.Fields, "\t") + "\n" )

  return nil

}

func (g *CGIRefVar) _strip_seqs(refseq []byte, altseq [][]byte) ([]byte, [][]byte) {
  r := []byte{}
  a := [][]byte{}
  for ii:=0; ii<len(altseq); ii++ { a = append(a, []byte{}) }

  for ii:=0; ii<len(refseq); ii++ {
    if refseq[ii]!='-' {
      r = append(r, refseq[ii])
    }
  }

  for ii:=0; ii<len(altseq); ii++ {
    for jj:=0; jj<len(altseq[ii]); jj++ {
      if altseq[ii][jj] != '-' {
        a[ii] = append(a[ii], altseq[ii][jj])
      }
    }
  }

  return r,a
}


func (g *CGIRefVar) PrintAltAlleles(vartype int, ref_start, ref_len int, refseq []byte, altseq [][]byte, out *bufio.Writer) error {

  varscorevaf := ""
  varscoreeaf := ""
  varfilter := ""
  haplink := ""
  xref := ""
  allelefreq := ""
  altcalls := ""

  ref,alt := g._strip_seqs(refseq, altseq)

  for strand:=0; strand<len(alt); strand++ {
    allele_str := fmt.Sprintf("%d", strand+1)

    cur_start := 0
    cur_len := len(ref)
    if cur_len > len(alt[strand]) { cur_len = len(alt[strand]) }

    for cur_len>0 {

      noc_pfx_len := 0
      for ii:=0; ii<cur_len; ii++ {
        if (alt[strand][cur_start+ii] != 'n') && (alt[strand][cur_start+ii] != 'N') { break }
        noc_pfx_len++
      }

      if noc_pfx_len>0 {

        //                           0   1   2   3   4   5   6   7   8   9   10  11  12  13  14  15
        out.WriteString(fmt.Sprintf("%d\t%d\t%s\t%s\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
          g.Locus, g.Ploidy, allele_str, g.ChromStr,
          ref_start+cur_start, ref_start+cur_start+noc_pfx_len,
          "no-call",
          refseq, "?",
          varscorevaf, varscoreeaf, varfilter, haplink, xref, allelefreq, altcalls))

        cur_start += noc_pfx_len
        cur_len -= noc_pfx_len

      }

      ref_pfx_len := 0
      for ii:=0; ii<cur_len; ii++ {
        if (ref[cur_start+ii]=='n') || (ref[cur_start+ii]=='N') || (ref[cur_start+ii]!=alt[strand][cur_start+ii]) { break }
        ref_pfx_len++
      }

      if ref_pfx_len>0 {

        //                           0   1   2   3   4   5   6   7   8   9   10  11  12  13  14  15
        out.WriteString(fmt.Sprintf("%d\t%d\t%s\t%s\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
          g.Locus, g.Ploidy, allele_str, g.ChromStr,
          ref_start+cur_start, ref_start+cur_start+ref_pfx_len,
          "ref",
          "=", "=",
          varscorevaf, varscoreeaf, varfilter, haplink, xref, allelefreq, altcalls))

        cur_start += ref_pfx_len
        cur_len -= ref_pfx_len

      }

      sub_pfx_len := 0
      for ii:=0; ii<cur_len; ii++ {
        if ref[cur_start+ii] == alt[strand][cur_start+ii] { break }
        sub_pfx_len++
      }

      if sub_pfx_len>0 {

        vartype_str := "snp"
        if sub_pfx_len > 1 { vartype_str = "sub" }

        //                           0   1   2   3   4   5   6   7   8   9   10  11  12  13  14  15
        out.WriteString(fmt.Sprintf("%d\t%d\t%s\t%s\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
          g.Locus, g.Ploidy, allele_str, g.ChromStr,
          ref_start+cur_start, ref_start+cur_start+sub_pfx_len,
          vartype_str,
          ref[cur_start:cur_start+sub_pfx_len],
          alt[strand][cur_start:cur_start+sub_pfx_len],
          varscorevaf, varscoreeaf, varfilter, haplink, xref, allelefreq, altcalls))

        cur_start += sub_pfx_len
        cur_len -= sub_pfx_len

      }

    }

    if cur_len>len(ref) {
      //                           0   1   2   3   4   5   6   7   8   9   10  11  12  13  14  15
      out.WriteString(fmt.Sprintf("%d\t%d\t%s\t%s\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
        g.Locus, g.Ploidy, allele_str, g.ChromStr,
        ref_start+cur_start, ref_start+cur_start+len(alt[strand]),
        "ins",
        "",
        alt[strand][cur_start:],
        varscorevaf, varscoreeaf, varfilter, haplink, xref, allelefreq, altcalls))

    } else if cur_len>len(alt[strand]) {

      //                           0   1   2   3   4   5   6   7   8   9   10  11  12  13  14  15
      out.WriteString(fmt.Sprintf("%d\t%d\t%s\t%s\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
        g.Locus, g.Ploidy, allele_str, g.ChromStr,
        ref_start+cur_start, ref_start+cur_start+len(ref),
        "del",
        ref[cur_start:],
        "",
        varscorevaf, varscoreeaf, varfilter, haplink, xref, allelefreq, altcalls))

    }

  }

  g.Locus++

  return nil
}

func (g *CGIRefVar) Print(vartype int, ref_start, ref_len int, refseq []byte, altseq [][]byte, out *bufio.Writer) error {

  if g.PrintHeader {
    g.Header(out)
    g.PrintHeader = false
  }

  ploidy := g.Ploidy
  allele_str := "all"
  chrom := g.ChromStr

  varscorevaf := ""
  varscoreeaf := ""
  varfilter := "PASS"
  haplink := ""
  xref := ""
  allelefreq := ""
  altcalls := ""

  vartype_str := "no-ref"

  ref,alt := g._strip_seqs(refseq, altseq) ;  _ = ref ; _ = alt

  if vartype == NOREF {
    vartype_str = "no-ref"

    //                           0   1   2   3   4   5   6   7   8   9   10  11  12  13  14  15
    out.WriteString(fmt.Sprintf("%d\t%d\t%s\t%s\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
      g.Locus, ploidy, allele_str, chrom,
      ref_start, ref_start+ref_len,
      vartype_str,
      "=", "?",
      varscorevaf, varscoreeaf, varfilter, haplink, xref, allelefreq, altcalls))

  } else if vartype == REF {
    vartype_str = "ref"

    //                           0   1   2   3   4   5   6   7   8   9   10  11  12  13  14  15
    out.WriteString(fmt.Sprintf("%d\t%d\t%s\t%s\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
      g.Locus, ploidy, allele_str, chrom,
      ref_start, ref_start+ref_len,
      vartype_str,
      "=", "=",
      varscorevaf, varscoreeaf, varfilter, haplink, xref, allelefreq, altcalls))

  } else if (vartype==NOC) || (vartype==ALT) {

    //snp, sub, ins, del

    g.PrintAltAlleles(vartype, ref_start, ref_len, refseq, altseq, out)

  }

  return nil

}

func (g *CGIRefVar) PrintEnd(out *bufio.Writer) error {
  out.Flush()
  return nil
}

//---

func (g *CGIRefVar) PastaBegin(out *bufio.Writer) error {
  g.Seq = make([][]byte, g.Ploidy)
  for ii:=0; ii<g.Ploidy; ii++ {
    g.Seq[ii] = make([]byte, 0, 1024)
  }

  g.Locus = 0

  return nil
}

func (g *CGIRefVar) PastaEnd(out *bufio.Writer) error {

  // Print out the sequence that we can,
  // first filling the the shorter of the
  // two sequences with the 'nop' character
  // '.' (period)
  //
  max_len := len(g.Seq[0])
  min_len := len(g.Seq[0])
  for a:=1; a<len(g.Seq); a++ {
    if max_len < len(g.Seq[a]) { max_len = len(g.Seq[a]) }
    if min_len > len(g.Seq[a]) { min_len = len(g.Seq[a]) }
  }

  for a:=0; a<len(g.Seq); a++ {
    for ii:=min_len; ii<max_len; ii++ {
      g.Seq[a] = append(g.Seq[a], '.')
    }
  }

  for ii:=0; ii<max_len; ii++ {
    for a:=0; a<len(g.Seq); a++ {
      out.WriteByte(g.Seq[a][ii])

      if (g.LFMod>0) && (g.OCounter > 0) && ((g.OCounter%g.LFMod)==0) {
        out.WriteByte('\n')
      }
      g.OCounter++

    }
  }

  for a:=0; a<len(g.Seq); a++ { g.Seq[a] = g.Seq[a][max_len:] }

  out.Flush()
  return nil
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

func (g *CGIRefVar) Pasta(cgivar_line string, ref_stream *bufio.Reader, out *bufio.Writer) error {
  if len(cgivar_line)==0 || cgivar_line[0] == '#' { return nil }

  str_allele := "2" ; _ = str_allele

  fields := strings.Split(cgivar_line, "\t")

  idx:=0
  locus,e := strconv.Atoi(fields[idx]) ; _ = locus
  if e!=nil { return e }

  idx++
  ploidy,e := strconv.Atoi(fields[idx]) ; _ = ploidy
  if e!=nil { return e }

  idx++
  allele := fields[idx] ; _ = allele

  idx++
  chrom := fields[idx] ; _ = chrom

  idx++
  _beg,e := strconv.Atoi(fields[idx]) ; _ = _beg
  if e!=nil { return e }

  idx++
  _end,e := strconv.Atoi(fields[idx]) ; _ = _end
  if e!=nil { return e }

  idx++
  vartype := fields[idx] ; _ = vartype

  idx++
  refseq := fields[idx] ; _ = refseq

  idx++
  alleleseq := fields[idx] ; _ = alleleseq

  dn := _end - _beg

  print_separator := false

  // Print out message if either the chromosome
  // or start position has changed
  //
  if chrom != g.ChromStr {
    out.WriteString(fmt.Sprintf(">C{%s}", chrom))
    g.ChromStr = chrom
    print_separator = true
  }
  if _beg != g.Start {
    out.WriteString(fmt.Sprintf(">P{%d}", _beg))
    print_separator = true
  }
  if ploidy != g.Ploidy {
    out.WriteString(fmt.Sprintf(">A{%d}", ploidy))
    g.Ploidy = ploidy
    print_separator = true
  }
  if print_separator { out.WriteByte('\n') }

  // Get list of sequence indices for updating
  // each of the output pasta streams.
  //

  map_idx := make(map[int]bool)
  for ii:=0; ii<g.Ploidy; ii++ {
    map_idx[ii] = false
  }

  seq_idx := []int{}
  if allele=="all" {
    for ii:=0; ii<g.Ploidy; ii++ {
      seq_idx = append(seq_idx, ii)
      map_idx[ii] = true
    }
  } else {
    z,e := strconv.Atoi(allele)
    if e!=nil { return e }
    seq_idx = append(seq_idx, z)
    map_idx[z] = true
  }

  nop_idx := []int{}
  for k,v := range map_idx {
    if !v {
      nop_idx = append(nop_idx, k)
    }
  }

  // Fill out previous locus if necessary
  //
  for locus!=g.Locus {
    g.Locus = locus

    // Simple case of single strand, print out PASTA stream
    // without issue
    //
    if len(g.Seq)==1 {
      for ii:=0; ii<len(g.Seq[0]); ii++ {
        out.WriteByte(g.Seq[0][ii])
        if (g.LFMod>0) && (g.OCounter > 0) && ((g.OCounter%g.LFMod)==0) {
          out.WriteByte('\n')
        }
        g.OCounter++
      }
      break
    }

    // We need to interleave the two 'raw' pasta streams
    // to insert the no-op character ('.') at the appropriate
    // locations.
    // The assumption is that the PASTA stream is ref-aligned
    // at every interleaved character group.
    //
    if len(g.Seq)==2 {
      a_pos := 0
      b_pos := 0

      for (a_pos<len(g.Seq[0])) && (b_pos<len(g.Seq[1])) {
        a_state := pasta.BPState[g.Seq[0][a_pos]]
        b_state := pasta.BPState[g.Seq[1][b_pos]]

        a_pasta_ch := byte('.')
        b_pasta_ch := byte('.')

        if (a_state == pasta.INS) || (a_state == pasta.DEL) {
          a_pasta_ch = g.Seq[0][a_pos]
          a_pos++
        } else if (b_state == pasta.INS) || (b_state == pasta.DEL) {
          b_pasta_ch = g.Seq[1][b_pos]
          b_pos++
        } else {
          a_pasta_ch = g.Seq[0][a_pos]
          b_pasta_ch = g.Seq[1][b_pos]
          a_pos++
          b_pos++
        }

        out.WriteByte(a_pasta_ch)
        out.WriteByte(b_pasta_ch)
      }

      for ; a_pos<len(g.Seq[0]); a_pos++ {
        out.WriteByte(g.Seq[0][a_pos])
        out.WriteByte('.')
      }

      for ; b_pos<len(g.Seq[0]); b_pos++ {
        out.WriteByte(g.Seq[0][b_pos])
        out.WriteByte('.')
      }

      break
    }

    return fmt.Errorf("Ploidy>2 not implemented")

  }


  // Case analysis for each type:
  //   no-ref, ref, no-call, snp, sub, ins, del
  //
  if vartype == "no-ref" {

    for ii:=0; ii<dn; ii++ {
      _,e := ref_stream.ReadByte()
      if e!=nil { return e }

      for a:=0; a<len(seq_idx); a++ {
        idx := seq_idx[a]
        g.Seq[idx] = append(g.Seq[idx], 'n')
      }

      //out.WriteByte('n')
    }

  } else if vartype == "no-call" {

    for ii:=0; ii<dn; ii++ {
      _,e := ref_stream.ReadByte()
      if e!=nil { return e }

      for a:=0; a<len(seq_idx); a++ {
        idx := seq_idx[a]
        g.Seq[idx] = append(g.Seq[idx], 'n')
      }

      //out.WriteByte('n')
    }

  } else if vartype == "ref" {

    for ii:=0; ii<dn; ii++ {
      ref_bp,e := ref_stream.ReadByte()
      if e!=nil { return e }

      for a:=0; a<len(seq_idx); a++ {
        idx := seq_idx[a]
        g.Seq[idx] = append(g.Seq[idx], ref_bp)
      }

      //out.WriteByte(ref_bp)
    }

  } else if vartype == "snp" {

    ref_bp,e := ref_stream.ReadByte()
    if e!=nil { return e }

    snp_bp := pasta.SubMap[ref_bp][alleleseq[0]]

    for a:=0; a<len(seq_idx); a++ {
      idx := seq_idx[a]
      g.Seq[idx] = append(g.Seq[idx], snp_bp)
    }

    //out.WriteByte(snp_bp)

  } else if vartype == "sub" {

    for ii:=0; ii<len(alleleseq); ii++ {
      ref_bp,e := ref_stream.ReadByte()
      if e!=nil { return e }

      pasta_ch := pasta.SubMap[ref_bp][_tolch(alleleseq[ii])]

      for a:=0; a<len(seq_idx); a++ {
        idx := seq_idx[a]
        g.Seq[idx] = append(g.Seq[idx], pasta_ch)
      }

      //out.WriteByte(pasta_ch)
    }

  } else if vartype == "ins" {
    //...

    for ii:=0; ii<len(alleleseq); ii++ {

      pasta_ch := pasta.InsMap[_tolch(alleleseq[ii])]

      for a:=0; a<len(seq_idx); a++ {
        idx := seq_idx[a]
        g.Seq[idx] = append(g.Seq[idx], pasta_ch)
      }

      //out.WriteByte(pasta_ch)
    }

  } else if vartype == "del" {
    //...

    for ii:=0; ii<len(alleleseq); ii++ {

      pasta_ch := pasta.DelMap[_tolch(alleleseq[ii])]

      for a:=0; a<len(seq_idx); a++ {
        idx := seq_idx[a]
        g.Seq[idx] = append(g.Seq[idx], pasta_ch)
      }

      //out.WriteByte(pasta_ch)
    }

  }

  return nil

}
