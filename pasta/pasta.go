package main

import "fmt"
import "os"
import "io"
import "runtime"
import "runtime/pprof"

import "github.com/abeconnelly/autoio"
import "github.com/codegangsta/cli"

import "github.com/abeconnelly/pasta"
import "github.com/abeconnelly/simplestream"

var VERSION_STR string = "0.1.0"
var gVerboseFlag bool

var gProfileFlag bool
var gProfileFile string = "pasta.pprof"

var gMemProfileFlag bool
var gMemProfileFile string = "pasta.mprof"

var gFullRefSeqFlag bool = true

var g_debug bool = false

func echo_stream(stream *simplestream.SimpleStream) {
  var e error
  var ch byte
  for ch,e = stream.Getc() ; e==nil ; ch,e = stream.Getc() {
    fmt.Printf("%c", ch)
  }
}

type VarDiff struct {
  Type      string
  RefStart  int
  RefLen    int
  RefSeq    string
  AltSeq    []string
}


func InterleaveStreamToVarDiff(stream *simplestream.SimpleStream, N ...int) ([]VarDiff, error) {
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
    ch0,e0 := stream.Getc()
    ch1,e1 := stream.Getc()

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

func interleave_to_diff(stream *simplestream.SimpleStream, w io.Writer) error {
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

  if g_debug { fmt.Printf("%v\n", pasta.RefDelBP) }

  for {
    is_ref0 := false
    is_ref1 := false
    ch0,e0 := stream.Getc()
    ch1,e1 := stream.Getc()

    stream0_pos++
    stream1_pos++

    if e0!=nil && e1!=nil { break }

    // special case: nop
    //
    if ch0=='.' && ch1=='.' { continue }

    dbp0 := pasta.RefDelBP[ch0]
    dbp1 := pasta.RefDelBP[ch1]

    if g_debug {
      fmt.Printf("\n")
      fmt.Printf(">>> ch0 %c (%d), ch1 %c (%d), dbp0 +%d, dbp1 +%d, ref0_len %d, ref1_len %d\n", ch0, ch0, ch1, ch1, dbp0, dbp1, ref0_len, ref1_len)
    }

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

      if bp_val,ok := pasta.AltMap[ch0] ; ok { alt0 = append(alt0, bp_val) }
      if bp_val,ok := pasta.AltMap[ch1] ; ok { alt1 = append(alt1, bp_val) }

      continue
    }

    // assert ch0==ch1 if they're both reference
    if is_ref0 && is_ref1 && ch0!=ch1 {
      return fmt.Errorf(fmt.Sprintf("ERROR: stream position (%d,%d), stream0 token %c (%d), stream1 token %c (%d)", stream0_pos, stream1_pos, ch0, ch0, ch1, ch1))
    }

    if !is_refn_cur && is_refn_prv {

      if gFullRefSeqFlag {
        w.Write( []byte(fmt.Sprintf("ref\t%d\t%d\t%s\n", ref_start, ref_start+ref0_len, refseq)) )
      } else {
        w.Write( []byte(fmt.Sprintf("ref\t%d\t%d\t.\n", ref_start, ref_start+ref0_len)) )
      }

      ref_start += ref0_len

      ref0_len=0
      ref1_len=0

      alt0 = alt0[0:0]
      alt1 = alt1[0:0]
      refseq = refseq[0:0]

    } else if is_refn_cur && !is_refn_prv {

      a0 := string(alt0)
      if len(a0) == 0 { a0 = "-" }

      a1 := string(alt1)
      if len(a1) == 0 { a1 = "-" }

      r := string(refseq)
      if len(r) == 0 { r = "-" }

      //w.Write( []byte(fmt.Sprintf("alt\t%d\t%d\t%s/%s;%s\n", ref_start, ref_start+ref0_len, alt0, alt1, refseq)) )
      w.Write( []byte(fmt.Sprintf("alt\t%d\t%d\t%s/%s;%s\n", ref_start, ref_start+ref0_len, a0, a1, r)) )

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

    if bp_val,ok := pasta.AltMap[ch0] ; ok { alt0 = append(alt0, bp_val) }
    if bp_val,ok := pasta.AltMap[ch1] ; ok { alt1 = append(alt1, bp_val) }

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

    is_refn_prv = is_refn_cur

  }

  // Final diff line
  //
  if is_refn_prv {
    if gFullRefSeqFlag {
      w.Write( []byte(fmt.Sprintf("ref\t%d\t%d\t%s\n", ref_start, ref_start+ref0_len, refseq)) )
    } else {
      w.Write( []byte(fmt.Sprintf("ref\t%d\t%d\t.\n", ref_start, ref_start+ref0_len)) )
    }
  } else if !is_refn_prv {

    a0 := string(alt0)
    if len(a0) == 0 { a0 = "-" }

    a1 := string(alt1)
    if len(a1) == 0 { a1 = "-" }

    r := string(refseq)
    if len(r) == 0 { r = "-" }

    w.Write( []byte(fmt.Sprintf("alt\t%d\t%d\t%s/%s;%s\n", ref_start, ref_start+ref0_len, a0, a1, r)) )
  }


  return nil
}

func interleave_streams(stream_a, stream_b *simplestream.SimpleStream, w io.Writer) error {
  var e0, e1 error
  ref_pos := [2]int{0,0}
  stm_pos := [2]int{0,0} ; _ = stm_pos
  ch_val := [2]byte{0,0}
  dot := [1]byte{'.'}

  for {

    if ref_pos[0] == ref_pos[1] {
      ch_val[0],e0 = stream_a.Getc()
      ch_val[1],e1 = stream_b.Getc()

      stm_pos[0]++
      stm_pos[1]++
    } else if ref_pos[0] < ref_pos[1] {
      ch_val[0],e0 = stream_a.Getc()

      stm_pos[0]++
    } else if ref_pos[0] > ref_pos[1] {
      ch_val[1],e1 = stream_b.Getc()

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
      w.Write(ch_val[0:2])
    } else if ref_pos[0] < ref_pos[1] {
      w.Write(ch_val[0:1])
      w.Write(dot[0:1])
    } else if ref_pos[0] > ref_pos[1] {
      w.Write(dot[0:1])
      w.Write(ch_val[1:2])
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

func _main( c *cli.Context ) {
  var e error
  action := "echo"

  infn_slice := c.StringSlice("input")

  stream    := simplestream.SimpleStream{}
  stream_b  := simplestream.SimpleStream{}

  g_debug = c.Bool("debug")

  gFullRefSeqFlag = c.Bool("full-sequence")

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
    stream.Init(fp)
  } else {
    fmt.Fprintf(os.Stderr, "Provide input stream")
    cli.ShowAppHelp(c)

    os.Exit(1)
  }

  if len(infn_slice)>1 {
    fp,e := os.Open(infn_slice[1])
    if e!=nil {
      fmt.Fprintf(os.Stderr, "%v", e)
      os.Exit(1)
    }
    defer fp.Close()
    stream_b.Init(fp)

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


  //---

  if action == "echo" {
    echo_stream(&stream)
  } else if action == "interleave" {
    interleave_streams(&stream, &stream_b, os.Stdout)
  } else if action == "rotini" {


    e:=interleave_to_diff(&stream, os.Stdout)
    if e!=nil { fmt.Fprintf(os.Stderr, "%v\n", e) ; return }


    //vardiff,e := InterleaveStreamToVarDiff(&stream)
    //if e!=nil { fmt.Fprintf(os.Stderr, "%v\n", e) ; return }
    //WriteVarDiff(vardiff, os.Stdout)

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
      Name: "action, a",
      Usage: "Action",
    },

    cli.BoolFlag{
      Name: "debug, d",
      Usage: "Debug",
    },

    cli.BoolFlag{
      Name: "full-sequence, F",
      Usage: "Display full sequence",
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
