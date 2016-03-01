package main

import "fmt"
import "bufio"

import "github.com/abeconnelly/pasta"

type PastaInfo struct {
  Stream *bufio.Reader
  StreamB *bufio.Reader
  Out *bufio.Writer

  RefPos int
  Ch []byte

  InsFlag bool
  DelFlag bool
  RefFlag bool
  NocFlag bool
}

func (g *PastaInfo) RotiniProcess() {

}

func pasta_filter(pasta_stream *bufio.Reader, out *bufio.Writer, start, n int) error {
  var msg ControlMessage ; _ = msg
  var pasta_stream_pos int
  var dbp int ; _ = dbp
  var curStreamState int ; _ = curStreamState

  var pasta_ref_pos int ; _ = pasta_ref_pos

  ref_pos:=0

  message_processed_flag := false
  for {

    //var ch1 byte

    ch,e := pasta_stream.ReadByte()
    for (e==nil) && ((ch=='\n') || (ch==' ') || (ch=='\r') || (ch=='\t')) {
      ch,e = pasta_stream.ReadByte()
    }
    if e!=nil { break }

    if ch=='>' {
      msg,e = process_control_message(pasta_stream)
      if e!=nil { return fmt.Errorf("invalid control message") }

      if (msg.Type == REF) || (msg.Type == NOC) {
        curStreamState = MSG
      } else if msg.Type==POS {
        ref_pos = msg.RefPos
      } else {

        //ignore
        //
        continue
      }

      control_message_print(msg, out)

      message_processed_flag = true
      continue
    }


    if message_processed_flag {
      out.WriteByte('\n')
    }

    message_processed_flag = false

    pasta_stream_pos++

    // special case: nop
    //
    if ch=='.'  { continue }

    dbp = pasta.RefDelBP[ch]

    //anch_bp := ch

    is_del := false ; _ = is_del
    is_ins := false ; _ = is_ins
    is_ref := false ; _ = is_ref
    is_noc := false ; _ = is_noc

    if ch=='!' || ch=='$' || ch=='7' || ch=='E' || ch=='z' {
      is_del = true
    } else if ch=='Q' || ch=='S' || ch=='W' || ch=='d' || ch=='Z' {
      is_ins = true
    } else if ch=='a' || ch=='c' || ch=='g' || ch=='t' {
      is_ref = true
    } else if ch=='n' || ch=='N' || ch == 'A' || ch == 'C' || ch == 'G' || ch == 'T' {
      is_noc = true
    }

    if (ref_pos >= start) && (ref_pos < (start+n)) {
      out.WriteByte(ch)
    }

    // Add to reference sequence
    //
    for {
      if is_ins { break }
      ref_pos++
      break
    }

  }

  return nil
}

func interleave_filter(pasta_stream *bufio.Reader, out *bufio.Writer, start, n int) error {
  var msg ControlMessage ; _ = msg
  var e error
  var e0 error
  var pasta_stream0_pos, pasta_stream1_pos int
  var dbp0,dbp1 int ; _,_ = dbp0,dbp1
  var curStreamState int ; _ = curStreamState

  var pasta_ref_pos int ; _ = pasta_ref_pos


  bp_count:=0
  lfmod := 50 ; _ = lfmod
  ref_pos:=0

  ch := [2]byte{}


  message_processed_flag := false

  for {

    //var ch1 byte
    var e1 error

    ch[0],e0 = pasta_stream.ReadByte()
    for (e0==nil) && ((ch[0]=='\n') || (ch[0]==' ') || (ch[0]=='\r') || (ch[0]=='\t')) {
      ch[0],e0 = pasta_stream.ReadByte()
    }
    if e0!=nil { break }

    if ch[0]=='>' {
      msg,e = process_control_message(pasta_stream)
      if e!=nil { return fmt.Errorf("invalid control message") }

      if (msg.Type == REF) || (msg.Type == NOC) {
        curStreamState = MSG
      } else if msg.Type==POS {
        ref_pos = msg.RefPos
      }

      control_message_print(msg, out)
      message_processed_flag = true
      continue
    }

    if message_processed_flag {
      out.WriteByte('\n')
    }
    message_processed_flag = false

    ch[1],e1 = pasta_stream.ReadByte()
    for (e1==nil) && ((ch[1]=='\n') || (ch[1]==' ') || (ch[1]=='\r') || (ch[1]=='\t')) {
      ch[1],e1 = pasta_stream.ReadByte()
    }
    if e1!=nil { break }

    pasta_stream0_pos++
    pasta_stream1_pos++

    // special case: nop
    //
    if ch[0]=='.' && ch[1]=='.' { continue }

    dbp0 = pasta.RefDelBP[ch[0]]
    dbp1 = pasta.RefDelBP[ch[1]]

    anch_bp := ch[0]
    if anch_bp == '.' { anch_bp = ch[1] }

    is_del := []bool{false,false}
    is_ins := []bool{false,false}
    is_ref := []bool{false,false} ; _ = is_ref
    is_noc := []bool{false,false} ; _ = is_noc

    for aa:=0; aa<2; aa++ {
      if ch[aa]=='!' || ch[aa]=='$' || ch[aa]=='7' || ch[aa]=='E' || ch[aa]=='z' {
        is_del[aa] = true
      } else if ch[aa]=='Q' || ch[aa]=='S' || ch[aa]=='W' || ch[aa]=='d' || ch[aa]=='Z' {
        is_ins[aa] = true
      } else if ch[aa]=='a' || ch[aa]=='c' || ch[aa]=='g' || ch[aa]=='t' {
        is_ref[aa] = true
      } else if ch[aa]=='n' || ch[aa]=='N' || ch[aa] == 'A' || ch[aa] == 'C' || ch[aa] == 'G' || ch[aa] == 'T' {
        is_noc[aa] = true
      }
    }


    if (is_ins[0] && (!is_ins[1] && ch[1]!='.')) ||
       (is_ins[1] && (!is_ins[0] && ch[0]!='.')) {
      return fmt.Errorf( fmt.Sprintf("insertion mismatch (ch %c,%c ord(%v,%v) @ %v)", ch[0], ch[1], ch[0], ch[1], bp_count) )
    }

    if (ref_pos >= start) && (ref_pos < (start+n)) {
      if ref_pos == start {
        out.WriteString(fmt.Sprintf(">P{%d}\n", ref_pos))
      }
      out.WriteByte(ch[0])
      out.WriteByte(ch[1])
    }


    // Add to reference sequence
    //
    for {
      if is_ins[0] || is_ins[1] { break }
      ref_pos++
      break
    }

  }

  return nil
}
