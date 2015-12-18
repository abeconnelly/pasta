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

func echo_stream(stream *simplestream.SimpleStream) {
  var e error
  var ch byte
  for ch,e = stream.Getc() ; e==nil ; ch,e = stream.Getc() {
    fmt.Printf("%c", ch)
  }
}

func interleave_to_diff(stream *simplestream.SimpleStream, w io.Writer) error {

  alt0 := []byte{}
  alt1 := []byte{}
  refseq := []byte{}

  prev_ref0 := true
  prev_ref1 := true

  ref_start := 0
  ref0_len := 0
  ref1_len := 0

  for {
    is_ref0 := false
    is_ref1 := false
    ch0,e0 := stream.Getc()
    ch1,e1 := stream.Getc()
    if e0!=nil && e1!=nil { break }

    //if (ch0!='Q') && (ch0!='S') && (ch0!='W') && (ch0!='d') && (ch0!='.') && (ch0!='\n') && (ch0!=' ') {
    if ch0=='a' || ch0=='c' || ch0=='g' || ch0=='t' || ch0=='n' {
      is_ref0=true
      ref0_len++
    }

    if ch1=='a' || ch1=='c' || ch1=='g' || ch1=='t' || ch1=='n' {
      is_ref1=true
      ref1_len++
    }

    if (!is_ref0 || !is_ref1) && prev_ref0 && prev_ref1 {
      //w.WriteString( fmt.Sprintf("ref\t%d\t%d\n", ref_start, ref_start+ref_len0) )
      w.Write( []byte(fmt.Sprintf("ref\t%d\t%d\n", ref_start, ref_start+ref0_len)) )

      ref_start += ref0_len

      ref0_len=0
      ref1_len=0

      alt0 = alt0[0:0]
      alt1 = alt1[0:0]
      refseq = refseq[0:0]
    } else if is_ref0 && is_ref1 && (!prev_ref0 || !prev_ref1) {
      w.Write( []byte(fmt.Sprintf("alt\t%d\t%d\t%s/%s;%s\n", ref_start, ref_start+ref0_len, alt0, alt1, refseq)) )

      ref_start += ref0_len

      alt0 = alt0[0:0]
      alt1 = alt1[0:0]
      refseq = refseq[0:0]
    }

    if !is_ref0 {
      bp_val := pasta.AltMap[ch0]
      alt0 = append(alt0, bp_val)
    }

    if !is_ref1 {
      bp_val := pasta.AltMap[ch1]
      alt1 = append(alt0, bp_val)
    }

    if !is_ref0 || !is_ref1 {
      if bp,ok := pasta.RefMap[ch0] ; ok {
        refseq = append(refseq, bp)
      } else if bp, ok := pasta.RefMap[ch1] ; ok {
        refseq = append(refseq, bp)
      }
    }

    prev_ref0 = is_ref0
    prev_ref1 = is_ref1

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


func _main( c *cli.Context ) {
  var e error
  action := "echo"

  infn_slice := c.StringSlice("input")

  stream    := simplestream.SimpleStream{}
  stream_b  := simplestream.SimpleStream{}

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
    interleave_to_diff(&stream, os.Stdout)
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
