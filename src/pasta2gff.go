package main

/*
*/

import "fmt"
import "io"
import "os"
import "runtime"
import "runtime/pprof"

import "bufio"
import "bytes"

import "github.com/abeconnelly/simplestream"

import "github.com/codegangsta/cli"

var VERSION_STR string = "0.1.0"
var gVerboseFlag bool

var gProfileFlag bool
var gProfileFile string = "pasta2gff.pprof"

var gMemProfileFlag bool
var gMemProfileFile string = "pasta2gff.mprof"

// Ref to Alt
//
var gSub map[byte]map[byte]byte
var gRefBP map[byte]byte
var gAltBP map[byte]byte
var gPastaBPState map[byte]int

func init() {

  gPastaBPState = make(map[byte]int)

  gSub = make(map[byte]map[byte]byte)

  gSub['a'] = make(map[byte]byte)
  gSub['a']['a'] = 'a'
  gSub['a']['c'] = '~'
  gSub['a']['g'] = '?'
  gSub['a']['t'] = '@'
  gSub['a']['n'] = 'A'

  gSub['c'] = make(map[byte]byte)
  gSub['c']['a'] = '='
  gSub['c']['c'] = 'c'
  gSub['c']['g'] = ':'
  gSub['c']['t'] = ';'
  gSub['c']['n'] = 'C'

  gSub['g'] = make(map[byte]byte)
  gSub['g']['a'] = '#'
  gSub['g']['c'] = '&'
  gSub['g']['g'] = 'g'
  gSub['g']['t'] = '%'
  gSub['g']['n'] = 'G'

  gSub['t'] = make(map[byte]byte)
  gSub['t']['a'] = '*'
  gSub['t']['c'] = '+'
  gSub['t']['g'] = '-'
  gSub['t']['t'] = 't'
  gSub['t']['n'] = 'T'

  gSub['n'] = make(map[byte]byte)
  gSub['n']['a'] = '\''
  gSub['n']['c'] = '"'
  gSub['n']['g'] = ','
  gSub['n']['t'] = '_'
  gSub['n']['n'] = 'n'

  gRefBP = make(map[byte]byte)
  gAltBP = make(map[byte]byte)
  gRefBP['a'] = 'a'
  gRefBP['~'] = 'a'
  gRefBP['?'] = 'a'
  gRefBP['@'] = 'a'
  gRefBP['A'] = 'a'

  gAltBP['a'] = 'a'
  gAltBP['~'] = 'c'
  gAltBP['?'] = 'g'
  gAltBP['@'] = 't'
  gAltBP['A'] = 'n'

  //-

  gRefBP['='] = 'c'
  gRefBP['c'] = 'c'
  gRefBP[':'] = 'c'
  gRefBP[';'] = 'c'
  gRefBP['C'] = 'c'

  gAltBP['='] = 'a'
  gAltBP['c'] = 'c'
  gAltBP[':'] = 'g'
  gAltBP[';'] = 't'
  gAltBP['C'] = 'n'

  //-

  gRefBP['#'] = 'g'
  gRefBP['&'] = 'g'
  gRefBP['g'] = 'g'
  gRefBP['%'] = 'g'
  gRefBP['G'] = 'g'

  gAltBP['#'] = 'a'
  gAltBP['&'] = 'c'
  gAltBP['g'] = 'g'
  gAltBP['%'] = 't'
  gAltBP['G'] = 'n'

  //-

  gRefBP['*'] = 't'
  gRefBP['+'] = 't'
  gRefBP['-'] = 't'
  gRefBP['t'] = 't'
  gRefBP['T'] = 't'

  gAltBP['*'] = 'a'
  gAltBP['+'] = 'c'
  gAltBP['-'] = 'g'
  gAltBP['t'] = 't'
  gAltBP['T'] = 'n'

  //-

  // Alt deletetions
  //
  gRefBP['!'] = 'a'
  gRefBP['$'] = 'c'
  gRefBP['7'] = 'g'
  gRefBP['E'] = 't'


  //--
  gPastaBPState['N'] = NOC
  gPastaBPState['n'] = NOC

  gPastaBPState['a'] = REF
  gPastaBPState['~'] = SUB
  gPastaBPState['?'] = SUB
  gPastaBPState['@'] = SUB
  gPastaBPState['A'] = NOC

  //-

  gPastaBPState['='] = SUB
  gPastaBPState['c'] = REF
  gPastaBPState[':'] = SUB
  gPastaBPState[';'] = SUB
  gPastaBPState['C'] = NOC

  //-

  gPastaBPState['#'] = SUB
  gPastaBPState['&'] = SUB
  gPastaBPState['g'] = REF
  gPastaBPState['%'] = SUB
  gPastaBPState['G'] = NOC

  //-

  gPastaBPState['*'] = SUB
  gPastaBPState['+'] = SUB
  gPastaBPState['-'] = SUB
  gPastaBPState['t'] = REF
  gPastaBPState['T'] = NOC

  //-

  gPastaBPState['!'] = INDEL
  gPastaBPState['$'] = INDEL
  gPastaBPState['7'] = INDEL
  gPastaBPState['E'] = INDEL

  gPastaBPState['Q'] = INDEL
  gPastaBPState['S'] = INDEL
  gPastaBPState['W'] = INDEL
  gPastaBPState['d'] = INDEL

}


const(
  REF = iota
  SNP = iota
  SUB = iota
  INDEL = iota
  NOC = iota
  FIN = iota
)

var g_CHROM string = "unk"
var g_INST string = "UNK"

// Position is 0 REFERENCE
// End is INCLUSIVE
//
func emit_ref(bufout *bufio.Writer, s,n int64) {
  bufout.WriteString( fmt.Sprintf("%s\t%s\tREF\t%d\t%d\t.\t+\t.\t.\n", g_CHROM, g_INST, s+1,s+n) )
}

// Position is 0 REFERENCE
// End is INCLUSIVE
//
func emit_sub_haploid(bufout *bufio.Writer, s,n int64, typ int, sub, ref []byte) {
  typ_str := ""
  if typ == SNP { typ_str = "SNP"
  } else if typ == SUB { typ_str = "SUB" }

  if len(sub)==0 { sub = []byte{'-'} }
  if len(ref)==0 { ref = []byte{'-'} }

  sub_str := fmt.Sprintf("alleles %s;ref_allele %s", sub, ref)

  bufout.WriteString( fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t.\t+\t.\t%s\n", g_CHROM, g_INST, typ_str, s+1, s+n, sub_str) )
}

// Position is 0 REFERENCE
// End is INCLUSIVE
//
func emit_indel_haploid(bufout *bufio.Writer, s,n int64, sub, ref []byte) {
  if len(sub)==0 { sub = []byte{'-'} }
  if len(ref)==0 { ref = []byte{'-'} }

  indel_str := fmt.Sprintf("alleles %s;ref_allele %s", sub, ref)

  bufout.WriteString( fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t.\t+\t.\t%s\n", g_CHROM, g_INST, "INDEL", s+1, s+n, indel_str) )
}

// Position is 0 REFERENCE
// End is INCLUSIVE
//
func emit_alt(bufout *bufio.Writer, s,n int64, typ int, altA, altB, ref []byte) {
  typ_str := ""
  switch typ {
  case REF: typ_str = "REF"
  case SNP: typ_str = "SNP"
  case SUB: typ_str = "SUB"
  case INDEL: typ_str = "INDEL"
  }
  if len(altA)==0 { altA = []byte{'-'} }
  if len(altB)==0 { altB = []byte{'-'} }
  if len(ref)==0 { ref = []byte{'-'} }

  if bytes.Equal(altA, altB) {
    bufout.WriteString( fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t.\t+\t.\talleles %s;ref_allele %s\n", g_CHROM, g_INST, typ_str, s+1, s+n, altA, ref) )
  } else {
    bufout.WriteString( fmt.Sprintf("%s\t%s\t%s\t%d\t%d\t.\t+\t.\talleles %s/%s;ref_allele %s\n", g_CHROM, g_INST, typ_str, s+1, s+n, altA, altB, ref) )
  }

}

func convert_diploid(ainA, ainB *simplestream.SimpleStream, aout *os.File, start_pos int64) error {
  var e error = nil
  var ok bool
  stream_pos:=-1

  alt0_seq := make([]byte, 0, 1024)
  alt1_seq := make([]byte, 0, 1024)
  ref_seq := make([]byte, 0, 1024)

  cur_state := REF
  next_state := -1
  next_state0 := -1
  next_state1 := -1

  cur_start := start_pos
  var cur_len int64 = 0

  allele_num := 0 ; _ = allele_num

  bufout := bufio.NewWriter(aout)
  defer bufout.Flush()

  bp0_ready := false
  bp1_ready := false

  var bp0 byte
  var bp1 byte

  for ;; {

    if !bp0_ready {
      if ainA.Pos>=ainA.N {

        if e=ainA.Refresh()
        e!=nil {
          next_state = FIN
          break
        }

      }

      bp0 = ainA.Buf[ainA.Pos]
      ainA.Pos++
      bp0_ready = true
      continue
    }

    if bp0 == ' ' || bp0 == '\n' {
      bp0_ready = false
      continue
    }

    if !bp1_ready {
      if ainB.Pos>=ainB.N {

        if e=ainB.Refresh()
        e!=nil {
          next_state = FIN
          break
        }

      }

      bp1 = ainB.Buf[ainB.Pos]
      ainB.Pos++
      bp1_ready=true
      continue
    }

    if bp1 == ' ' || bp1 == '\n' {
      bp1_ready = false
      continue
    }

    bp0_ready = false
    bp1_ready = false


    stream_pos++
    cur_len++

    next_state0,ok = gPastaBPState[bp0]
    if !ok {
      return fmt.Errorf("Invalid character (%c) at %d", bp0, stream_pos)
    }

    next_state1,ok = gPastaBPState[bp1]
    if !ok {
      return fmt.Errorf("Invalid character (%c) at %d", bp1, stream_pos)
    }

    if next_state0 == REF && next_state1 == REF {
      next_state = REF
    } else if next_state0 == INDEL || next_state1 == INDEL {
      next_state = INDEL
    } else if next_state0 == SUB || next_state1  == SUB {
      next_state = SUB
    } else if next_state0 == NOC || next_state1 == NOC {
      next_state = NOC
    }

    if cur_state == REF {
      if next_state != REF {
        emit_ref(bufout, cur_start, cur_len-1)

        cur_start += cur_len-1
        cur_len = 1
        cur_state = next_state
        ref_seq = ref_seq[0:0]
        alt0_seq = alt0_seq[0:0]
        alt1_seq = alt1_seq[0:0]
      }
    } else if cur_state == SUB {
      if next_state == INDEL {
        cur_state = INDEL
      } else if next_state == NOC  || next_state == REF {
        if len(alt0_seq)==1 && len(ref_seq)==1 { cur_state = SNP }
        emit_alt(bufout, cur_start, cur_len-1, cur_state, alt0_seq, alt1_seq, ref_seq)

        cur_start += cur_len-1
        cur_len = 1
        cur_state = next_state
        ref_seq = ref_seq[0:0]
        alt0_seq = alt0_seq[0:0]
        alt1_seq = alt1_seq[0:0]
      }
    } else if cur_state == INDEL {
      if next_state == INDEL || next_state == SNP || next_state == SUB {
      } else if next_state == REF || next_state == NOC {
        emit_alt(bufout, cur_start, cur_len-1, INDEL, alt0_seq, alt1_seq, ref_seq)

        cur_start += cur_len-1
        cur_len = 1
        ref_seq = ref_seq[0:0]
        alt0_seq = alt0_seq[0:0]
        alt1_seq = alt1_seq[0:0]
        cur_state = next_state
      }
    } else if cur_state == NOC {
      cur_start += cur_len-1
      cur_len = 1
      cur_state = next_state
    }

    if r,ok := gRefBP[bp0] ; ok {
      ref_seq = append(ref_seq, r)
    }

    if gRefBP[bp0] != gRefBP[bp1] {
      return fmt.Errorf( fmt.Sprintf("ref bases do not match at pos %d (%c != %c)", stream_pos, gRefBP[bp0], gRefBP[bp1]))
    }

    if r,ok := gAltBP[bp0] ; ok {
      alt0_seq = append(alt0_seq, r)
    }

    if r,ok := gAltBP[bp1] ; ok {
      alt1_seq = append(alt1_seq, r)
    }

  }

  if cur_state == REF {
    if next_state != REF {
      emit_ref(bufout, cur_start, cur_len)
    }
  } else if cur_state == SUB {
    if next_state == INDEL {
      cur_state = INDEL
    } else if next_state == NOC  || next_state == REF || next_state == FIN {
      if len(alt0_seq)==1 && len(ref_seq)==1 { cur_state = SNP }
      emit_alt(bufout, cur_start, cur_len, cur_state, alt0_seq, alt1_seq, ref_seq)
    }
  } else if cur_state == INDEL {
    if next_state == INDEL || next_state == SNP || next_state == SUB {
    } else if next_state == REF || next_state == NOC || next_state == FIN {
      emit_alt(bufout, cur_start, cur_len, INDEL, alt0_seq, alt1_seq, ref_seq)
    }
  } else if cur_state == NOC {
    cur_start += cur_len
    cur_len = 0
    cur_state = next_state
  }


  return e
}

func convert_haploid(ain *simplestream.SimpleStream, aout *os.File, start_pos int64) error {
  var e error = nil
  var ok bool
  stream_pos:=-1

  alt_seq := make([]byte, 0, 1024)
  ref_seq := make([]byte, 0, 1024)

  cur_state := REF
  next_state := -1

  cur_start := start_pos
  var cur_len int64 = 0

  allele_num := 0 ; _ = allele_num

  bufout := bufio.NewWriter(aout)
  defer bufout.Flush()

  for ;; {

    if ain.Pos>=ain.N {

      if e=ain.Refresh()
      e!=nil {
        next_state = FIN
        break
      }

    }

    bp := ain.Buf[ain.Pos]
    ain.Pos++

    if bp == ' ' || bp == '\n' { continue }

    stream_pos++
    cur_len++

    next_state,ok = gPastaBPState[bp]
    if !ok {
      return fmt.Errorf("Invalid character (%c) at %d", bp, stream_pos)
    }

    if cur_state == REF {
      if next_state != REF {
        emit_ref(bufout, cur_start, cur_len-1)

        cur_start += cur_len-1
        cur_len = 1
        cur_state = next_state
        ref_seq = ref_seq[0:0]
        alt_seq = alt_seq[0:0]
      }
    } else if cur_state == SUB {
      if next_state == INDEL {
        cur_state = INDEL
      } else if next_state == NOC  || next_state == REF {
        if len(alt_seq)==1 && len(ref_seq)==1 { cur_state = SNP }
        emit_alt(bufout, cur_start, cur_len-1, cur_state, alt_seq, alt_seq, ref_seq)

        cur_start += cur_len-1
        cur_len = 1
        cur_state = next_state
        ref_seq = ref_seq[0:0]
        alt_seq = alt_seq[0:0]
      }
    } else if cur_state == INDEL {
      if next_state == INDEL || next_state == SNP || next_state == SUB {
      } else if next_state == REF || next_state == NOC {
        emit_alt(bufout, cur_start, cur_len-1, INDEL, alt_seq, alt_seq, ref_seq)

        cur_start += cur_len-1
        cur_len = 1
        ref_seq = ref_seq[0:0]
        alt_seq = alt_seq[0:0]
        cur_state = next_state
      }
    } else if cur_state == NOC {
      cur_start += cur_len-1
      cur_len = 1
      cur_state = next_state
    }

    if r,ok := gRefBP[bp] ; ok {
      ref_seq = append(ref_seq, r)
    }

    if r,ok := gAltBP[bp] ; ok {
      alt_seq = append(alt_seq, r)
    }

  }

  if cur_state == REF {
    if next_state != REF {
      emit_ref(bufout, cur_start, cur_len)
    }
  } else if cur_state == SUB {
    if next_state == INDEL {
      cur_state = INDEL
    } else if next_state == NOC  || next_state == REF || next_state == FIN {
      if len(alt_seq)==1 && len(ref_seq)==1 { cur_state = SNP }
      emit_alt(bufout, cur_start, cur_len, cur_state, alt_seq, alt_seq, ref_seq)
    }
  } else if cur_state == INDEL {
    if next_state == INDEL || next_state == SNP || next_state == SUB {
    } else if next_state == REF || next_state == NOC || next_state == FIN {
      emit_alt(bufout, cur_start, cur_len, INDEL, alt_seq, alt_seq, ref_seq)
    }
  } else if cur_state == NOC {
    cur_start += cur_len
    cur_len = 0
    cur_state = next_state
  }


  return e
}

func _main(c *cli.Context) {
  var err error

  /*
  if c.String("input") == "" {
    fmt.Fprintf( os.Stderr, "Input required, exiting\n" )
    cli.ShowAppHelp( c )
    os.Exit(1)
  }
  */

  infn_slice := c.StringSlice("input")
  if len(infn_slice)==0 {
    fmt.Fprintf( os.Stderr, "Input required, exiting\n" )
    cli.ShowAppHelp( c )
    os.Exit(1)
  }

  ain_count:=1

  ain := simplestream.SimpleStream{}
  fp := os.Stdin
  //if c.String("input") != "-" {
  if infn_slice[0] != "-" {
    var e error
    //fp ,e = os.Open(c.String("input"))
    fp ,e = os.Open(infn_slice[0])
    if e!=nil {
      fmt.Fprintf(os.Stderr, "%v", e)
      os.Exit(1)
    }
    defer fp.Close()
  }
  ain.Init(fp)

  ain2 := simplestream.SimpleStream{}

  if len(infn_slice)>1 {
    ain_count++

    fp2,e := os.Open(infn_slice[1])
    if e!=nil {
      fmt.Fprintf(os.Stderr, "%v", e)
      os.Exit(1)
    }
    defer fp2.Close()

    ain2.Init(fp)
  }

  var ref_start int64
  ref_start = 0
  ss := c.Int("ref-start")
  if ss > 0 { ref_start = int64(ss) }

  var seq_start int64
  seq_start = 0 ; _ = seq_start
  ss = c.Int("seq-start")
  if ss > 0 { seq_start = int64(ss) }

  aout := os.Stdout
  if c.String("output") != "-" {
    aout,err = os.Open(c.String("output"))
    if err!=nil {
      fmt.Fprintf(os.Stderr, "%v", err)
      os.Exit(1)
    }
    defer aout.Close()
  }


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

  //convert(&gff_ain, &ref_ain, aout.Fp, ref_start)
  if ain_count == 1 {
    e := convert_haploid(&ain, aout, ref_start)
    if e!=nil && e!=io.EOF { panic(e) }
  } else {
    e := convert_diploid(&ain, &ain2, aout, ref_start)
    if e!=nil && e!=io.EOF { panic(e) }
  }

}

func main() {

  app := cli.NewApp()
  app.Name  = "pasta2gff"
  app.Usage = "pasta2gff"
  app.Version = VERSION_STR
  app.Author = "Curoverse, Inc."
  app.Email = "info@curoverse.com"
  app.Action = func( c *cli.Context ) { _main(c) }

  app.Flags = []cli.Flag{
    /*
    cli.StringFlag{
      Name: "input, i",
      Usage: "INPUT",
    },
    */

    cli.StringSliceFlag{
      Name: "input, i",
      Usage: "INPUT",
    },

    cli.StringFlag{
      Name: "ref-input, r",
      Usage: "REF-INPUT",
    },

    cli.StringFlag{
      Name: "output, o",
      Value: "-",
      Usage: "OUTPUT",
    },

    cli.IntFlag{
      Name: "max-procs, N",
      Value: -1,
      Usage: "MAXPROCS",
    },

    cli.IntFlag{
      Name: "ref-start, S",
      Value: -1,
      Usage: "Start of reference stream (default to start of GFF position)",
    },

    cli.IntFlag{
      Name: "seq-start, s",
      Value: -1,
      Usage: "Start of reference stream (default to start of GFF position)",
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
