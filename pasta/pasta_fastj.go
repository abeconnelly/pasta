package main

// Convert a pasta stream to FastJ

//import "os"

import "fmt"
import "strconv"
import "strings"
import "bufio"
//import "io"
//import "bytes"

//import "time"

import "github.com/abeconnelly/pasta"

import "crypto/md5"

import "io"


type FastJHeader struct {
  TileID string
  Md5Sum string

  Locus []map[string]string

  N int
  SeedTileLength int

  StartTile bool
  EndTile bool

  StartSeq string
  StartTag string

  EndSeq string
  EndTag string

  NoCallCount int
  Notes []string
}

type FastJInfo struct {
  TagPath int
  TagStep int
  EndTagBuffer []string
  TagStream *bufio.Reader
  TagFinished bool

  AssemblyRef string
  AssemblyChrom string
  AssemblyPath int
  AssemblyStep int
  AssemblyPrevStep int
  AssemblyEndPos int
  AssemblyPrevEndPos int
  AssemblyStream *bufio.Reader

  RefTile []byte
  AltTile [][]byte

  LibraryVersion int

  RefPos int

  RefBuild string
  Chrom string
}

func (g *FastJInfo) Init() {
  g.EndTagBuffer = make([]string, 0, 2)
  g.TagFinished = false

  g.RefTile = make([]byte, 0, 1024)
  g.AltTile = make([][]byte, 2)
  g.AltTile[0] = make([]byte, 0, 1024)
  g.AltTile[1] = make([]byte, 0, 1024)

  g.RefPos=0
  g.LibraryVersion = 0
}

//--

func (g *FastJInfo) ReadTag(tag_stream *bufio.Reader) error {

  is_eof := false

  if g.TagFinished {
    return fmt.Errorf("tag stream finished")
  }

  for {
    l,e := tag_stream.ReadString('\n')

    if e!=nil { g.TagFinished = true }

    if e==io.EOF {
      is_eof=true
    } else if e!=nil {
      return e
    }

    if len(l)==0 {
      if is_eof { return io.EOF }
      continue
    }

    if l[0]=='>' {

      path_ver_str := l[1:]
      parts := strings.Split(path_ver_str, ".")
      _path,e := strconv.ParseUint(parts[0], 16, 64)
      if e!=nil { return e }

      g.TagPath = int(_path)
      g.TagStep = 0

      if is_eof { return io.EOF }
      continue
    }

    g.EndTagBuffer = append(g.EndTagBuffer, strings.Trim(l, " \t\n"))
    return nil
  }

}

//--

func (g *FastJInfo) ReadAssembly(assembly_stream *bufio.Reader) error {

  for {
    l,e := assembly_stream.ReadString('\n')
    if e!=nil { return e }
    if len(l)==0 { continue }

    if l[0]=='>' {

      ref_chr_path := strings.Trim(l, " \t>\n")

      parts := strings.Split(ref_chr_path, ":")
      ref_str := parts[0]
      chrom_str := parts[1]
      _path,e := strconv.ParseUint(parts[2], 16, 64)
      if e!=nil {
        return fmt.Errorf(fmt.Sprintf("ERROR: ReadAssembly: line '%s' part '%s': %v", l, parts[2], e))
      }

      g.AssemblyRef = ref_str
      g.AssemblyChrom = chrom_str
      g.AssemblyPath = int(_path)
      g.AssemblyStep = 0
      g.AssemblyPrevStep = 0
      continue
    }

    l_trim := strings.Trim(l, " \t\n")

    parts := strings.Split(l_trim, "\t")
    p0 := strings.Trim(parts[0], " \t\n")
    _step,e := strconv.ParseUint(p0, 16, 64)
    if e!=nil {
      return fmt.Errorf(fmt.Sprintf("ERROR: ReadAssembly: line '%s' part '%s': %v", l, parts[0], e))
    }

    p1 := strings.Trim(parts[1], "\t \n")
    _pos,e := strconv.ParseUint(p1, 10, 64)
    if e!=nil {
      return fmt.Errorf(fmt.Sprintf("ERROR: ReadAssembly: line '%s' part '%s': %v", l, parts[1], e))
    }

    g.AssemblyPrevEndPos = g.AssemblyEndPos
    g.AssemblyEndPos = int(_pos)
    g.AssemblyPrevStep = g.AssemblyStep
    g.AssemblyStep = int(_step)

    return nil
  }

}

func (g *FastJInfo) DebugPrint() {
  fmt.Printf("\n")
  fmt.Printf("\n")

  fmt.Printf("Assembly:\n")
  fmt.Printf("  Ref:   %s\n", g.AssemblyRef)
  fmt.Printf("  Chrom: %s\n", g.AssemblyChrom)
  fmt.Printf("  Path:     %x (%dd)\n", g.AssemblyPath, g.AssemblyPath)
  fmt.Printf("  PrevStep: %x (%dd)\n", g.AssemblyPrevStep, g.AssemblyPrevStep)
  fmt.Printf("  Step:     %x (%dd)\n", g.AssemblyStep, g.AssemblyStep)
  fmt.Printf("  PrevEndPos:  %d\n", g.AssemblyPrevEndPos)
  fmt.Printf("  EndPos:      %d\n", g.AssemblyEndPos)
  fmt.Printf("\n")

  fmt.Printf("Tag:\n")
  fmt.Printf("  TagPath: %x (%dd)\n", g.TagPath, g.TagPath)
  fmt.Printf("  TagStep: %x (%dd)\n", g.TagStep, g.TagStep)
  fmt.Printf("  EndTagBuffer:\n")
  for ii:=0; ii<len(g.EndTagBuffer); ii++ {
    fmt.Printf("    [%d] %s\n", ii, g.EndTagBuffer[ii])
  }
  fmt.Printf("\n")



}


func (g *FastJInfo) WriteFastJSeq(seq []byte, out *bufio.Writer) {
  w := 50

  q := len(seq)/w
  r := len(seq)%w

  for ii:=0; ii<q; ii++ {
    out.Write(seq[ii*w:(ii+1)*w])
    out.WriteByte('\n')
  }
  if r>0 {
    out.Write(seq[q*w:])
    out.WriteByte('\n')
  }

}


func _tf_val(v bool) string {
  if v {
    return "true"
  }
  return "false"
}

func _m5sum_str(b []byte) string {
  dat := md5.Sum(b)
  z := make([]string, 0, len(dat))
  for ii:=0; ii<len(dat); ii++ {
    z = append(z, fmt.Sprintf("%02x", dat[ii]))
  }
  return strings.Join(z, "")
}

func _noc_count(b []byte) int {
  c:=0
  for ii:=0; ii<len(b); ii++ {
    if (b[ii]=='n') || (b[ii]=='N') {
      c++
    }
  }
  return c
}

func (g *FastJInfo) EndTagMatch(seq []byte) bool {
  idx_end := len(g.EndTagBuffer)-1
  if idx_end<0 { return false }

  n := len(seq)
  if n<24 { return false }

  for ii:=0; ii<24; ii++ {
    if (seq[n-24+ii] == 'n') || (seq[n-24+ii] == 'N') { continue }
    if seq[n-24+ii] != g.EndTagBuffer[idx_end][ii] {
      return false
    }
  }

  return true

}

//--

func (g *FastJInfo) Convert(pasta_stream *bufio.Reader, tag_stream *bufio.Reader, assembly_stream *bufio.Reader, out *bufio.Writer) error {
  var msg ControlMessage ; _ = msg
  var e error
  var pasta_stream0_pos, pasta_stream1_pos int
  var dbp0,dbp1 int ; _,_ = dbp0,dbp1
  var curStreamState int ; _ = curStreamState

  ref_seq := make([]byte, 0, 1024)
  alt_seq := make([][]byte, 2)
  alt_seq[0] = make([]byte, 0, 1024)
  alt_seq[1] = make([]byte, 0, 1024)

  seed_tile_length := make([]int, 2)
  seed_tile_length[0] = 1
  seed_tile_length[1] = 1

  step_pos := make([]int, 2)
  step_pos[0] = 0
  step_pos[1] = 0

  lfmod := 50 ; _ = lfmod
  ref_pos:=g.RefPos

  e = g.ReadAssembly(assembly_stream)
  if e!=nil { return e }


  message_processed_flag := false ; _ = message_processed_flag
  for {

    var ch1 byte
    var e1 error

    ch0,e0 := pasta_stream.ReadByte()
    for (e0==nil) && ((ch0=='\n') || (ch0==' ') || (ch0=='\r') || (ch0=='\t')) {
      ch0,e0 = pasta_stream.ReadByte()
    }
    if e0!=nil { break }

    if ch0=='>' {
      msg,e = process_control_message(pasta_stream)
      if e!=nil { return fmt.Errorf("invalid control message") }

      if (msg.Type == REF) || (msg.Type == NOC) {
        curStreamState = MSG
      } else {

        //ignore
        //
        continue
      }

      message_processed_flag = true
      continue
    }


    // emit tiles
    //
    if ref_pos == g.AssemblyEndPos {

      end_tile_flag := false

      if !g.TagFinished {
        e = g.ReadTag(tag_stream)
        if e!=nil {
          return fmt.Errorf(fmt.Sprintf("ERROR reading tag: %v", e))
        }
      } else {
        end_tile_flag = true
      }

      s_epos := 24
      if s_epos > len(alt_seq[0]) { s_epos = len(alt_seq[0]) }

      e_spos := len(alt_seq[0])
      if e_spos < 0 { e_spos=0 }

      if end_tile_flag || g.EndTagMatch(alt_seq[0]) {

        start_tile_flag := false
        beg_tag := ""
        idx_end := len(g.EndTagBuffer)-1
        if (idx_end-seed_tile_length[0])>=0 {
          beg_tag = g.EndTagBuffer[idx_end-seed_tile_length[0]]
        } else {
          start_tile_flag = true
        }

        end_tag := ""
        if !end_tile_flag {
          end_tag = g.EndTagBuffer[idx_end]
        }

        out.WriteString(fmt.Sprintf(`>{"tileID":"%04x.%02x.%04x.%03x"`,
          g.TagPath, g.LibraryVersion, step_pos[0], 0))
        out.WriteString(fmt.Sprintf(`,"md5sum":"%s"`, _m5sum_str(alt_seq[0])))
        out.WriteString(fmt.Sprintf(`,"locus":[{"build":"%s %s %d %d"}]`, g.RefBuild, g.Chrom, g.AssemblyPrevEndPos, g.AssemblyEndPos))
        out.WriteString(fmt.Sprintf(`,"n":%d`, len(alt_seq[0])))
        out.WriteString(fmt.Sprintf(`,"seedTileLength":%d`, seed_tile_length[0]))
        out.WriteString(fmt.Sprintf(`,"startTile":%s`, _tf_val(start_tile_flag)))
        out.WriteString(fmt.Sprintf(`,"endTile":%s`, _tf_val(end_tile_flag)))
        out.WriteString(fmt.Sprintf(`,"startSeq":"%s","endSeq":"%s"`,
          alt_seq[0][0:s_epos],
          alt_seq[0][e_spos:]))
        out.WriteString(fmt.Sprintf(`,"startTag":"%s"`, beg_tag))
        out.WriteString(fmt.Sprintf(`,"endTag":"%s"`, end_tag))


        out.WriteString(fmt.Sprintf(`,"nocallCount":%d`, _noc_count(alt_seq[0])))
        out.WriteString(fmt.Sprintf(`,"notes":[]`))
        out.WriteString(fmt.Sprintf("}\n"))

        g.WriteFastJSeq(alt_seq[0], out)
        out.WriteByte('\n')

        // Update sequence
        //
        if len(alt_seq[0]) >= 24 {
          n:=len(alt_seq[0])
          alt_seq[0] = alt_seq[0][n-24:]
        }
        step_pos[0]+=seed_tile_length[0]

        seed_tile_length[0]=1

      } else {
        seed_tile_length[0]++
      }

      //----

      s_epos = 24
      if s_epos > len(alt_seq[1]) { s_epos = len(alt_seq[1]) }

      e_spos = len(alt_seq[1])-24
      if e_spos < 0 { e_spos=1 }

      if end_tile_flag || g.EndTagMatch(alt_seq[1]) {

        start_tile_flag := false
        beg_tag := ""
        idx_end := len(g.EndTagBuffer)-1
        if (idx_end-seed_tile_length[1])>=0 {
          beg_tag = g.EndTagBuffer[idx_end-seed_tile_length[1]]
        } else {
          start_tile_flag = true
        }

        end_tag := ""
        if !end_tile_flag {
          end_tag = g.EndTagBuffer[idx_end]
        }


        out.WriteString(fmt.Sprintf(`>{"tileID":"%04x.%02x.%04x.%03x"`,
          g.TagPath, g.LibraryVersion, step_pos[1], 1))
        out.WriteString(fmt.Sprintf(`,"md5sum":"%s"`, _m5sum_str(alt_seq[1])))
        out.WriteString(fmt.Sprintf(`,"locus":[{"build":"%s %s %d %d"}]`, g.RefBuild, g.Chrom, g.AssemblyPrevEndPos, g.AssemblyEndPos))
        out.WriteString(fmt.Sprintf(`,"n":%d`, len(alt_seq[0])))
        out.WriteString(fmt.Sprintf(`,"seedTileLength":%d`, seed_tile_length[1]))
        out.WriteString(fmt.Sprintf(`,"startTile":%s`, _tf_val(start_tile_flag)))
        out.WriteString(fmt.Sprintf(`,"endTile":%s`, _tf_val(end_tile_flag)))
        out.WriteString(fmt.Sprintf(`,"startSeq":"%s","endSeq":"%s"`,
          alt_seq[1][0:s_epos],
          alt_seq[1][e_spos:]))

        out.WriteString(fmt.Sprintf(`,"startTag":"%s"`, beg_tag))
        out.WriteString(fmt.Sprintf(`,"endTag":"%s"`, end_tag))

        out.WriteString(fmt.Sprintf(`,"nocallCount":%d`, _noc_count(alt_seq[1])))
        out.WriteString(fmt.Sprintf(`,"notes":[]`))
        out.WriteString(fmt.Sprintf("}\n"))

        g.WriteFastJSeq(alt_seq[1], out)
        out.WriteByte('\n')

        // Update sequence
        //
        if len(alt_seq[1]) >= 24 {
          n:=len(alt_seq[1])
          alt_seq[1] = alt_seq[1][n-24:]
        }
        step_pos[1]+=seed_tile_length[1]

        seed_tile_length[1]=1

      } else {
        seed_tile_length[1]++
      }

      if len(ref_seq) >= 24 {
        n := len(ref_seq)
        ref_seq = ref_seq[n-24:]
      }

      e = g.ReadAssembly(assembly_stream)
      if e!=nil { return fmt.Errorf(fmt.Sprintf("ERROR reading assembly: %v", e)) }

    }

    message_processed_flag = false

    ch1,e1 = pasta_stream.ReadByte()
    for (e1==nil) && ((ch1=='\n') || (ch1==' ') || (ch1=='\r') || (ch1=='\t')) {
      ch1,e1 = pasta_stream.ReadByte()
    }
    if e1!=nil { break }

    pasta_stream0_pos++
    pasta_stream1_pos++


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

    if (is_ins[0] && (!is_ins[1] && ch1!='.')) ||
       (is_ins[1] && (!is_ins[0] && ch0!='.')) {
      return fmt.Errorf( fmt.Sprintf("insertion mismatch (ch %c,%c ord(%v,%v) @ %v)", ch0, ch1, ch0, ch1, ref_pos) )
    }

    // Add to reference sequence
    //
    for {

      if is_ins[0] || is_ins[1] { break }
      if ch1 == '.' {
        ref_seq = append(ref_seq, pasta.RefMap[ch0])
      } else if ch0 == '.' {
        ref_seq = append(ref_seq, pasta.RefMap[ch1])
      } else {
        ref_bp := pasta.RefMap[ch0]
        if ref_bp != pasta.RefMap[ch1] {
          return fmt.Errorf( fmt.Sprintf("PASTA reference bases do not match (%c != %c) at %d %d (refpos %d)\n",
            ref_bp, pasta.RefMap[ch1], pasta_stream0_pos, pasta_stream1_pos, ref_pos) )
        }
        ref_seq = append(ref_seq, ref_bp)
      }
      ref_pos++
      break
    }

    // Alt sequences
    //
    for {
      if ch0=='.' { break }
      if pasta.IsAltDel[ch0] { break }
      alt_seq[0] = append(alt_seq[0], pasta.AltMap[ch0])
      break
    }

    for {
      if ch1=='.' { break }
      if pasta.IsAltDel[ch1] { break }
      alt_seq[1] = append(alt_seq[1], pasta.AltMap[ch1])
      break
    }

  }

  // ALT0
  //
  for aa:=0; aa<2; aa++ {

    start_tile_flag := false
    beg_tag := ""
    idx_end := len(g.EndTagBuffer)-1
    if (idx_end-seed_tile_length[aa])>=0 {
      beg_tag = g.EndTagBuffer[idx_end-seed_tile_length[aa]]
    } else {
      start_tile_flag = true
    }

    // We're at the end of the path, so no end tag
    //
    end_tag := ""

    s_epos := 24
    if s_epos > len(alt_seq[aa]) { s_epos = len(alt_seq[aa]) }

    e_spos := len(alt_seq[aa])-24
    if e_spos < 0 { e_spos=1 }


    out.WriteString(fmt.Sprintf(`>{"tileID":"%04x.%02x.%04x.%03x"`,
      g.TagPath, g.LibraryVersion, step_pos[aa], aa))
    out.WriteString(fmt.Sprintf(`,"md5sum":"%s"`, _m5sum_str(alt_seq[aa])))
    out.WriteString(fmt.Sprintf(`,"locus":[{"build":"%s %s %d %d"}]`, g.RefBuild, g.Chrom, g.AssemblyPrevEndPos, g.AssemblyEndPos))
    out.WriteString(fmt.Sprintf(`,"n":%d`, len(alt_seq[aa])))
    out.WriteString(fmt.Sprintf(`,"seedTileLength":%d`, seed_tile_length[aa]))
    out.WriteString(fmt.Sprintf(`,"startTile":%s`, _tf_val(start_tile_flag)))
    out.WriteString(fmt.Sprintf(`,"endTile":%s`, _tf_val(true)))
    out.WriteString(fmt.Sprintf(`,"startSeq":"%s","endSeq":"%s"`,
      alt_seq[0][0:s_epos],
      alt_seq[0][e_spos:]))
    out.WriteString(fmt.Sprintf(`,"startTag":"%s"`, beg_tag))
    out.WriteString(fmt.Sprintf(`,"endTag":"%s"`, end_tag))


    out.WriteString(fmt.Sprintf(`,"nocallCount":%d`, _noc_count(alt_seq[aa])))
    out.WriteString(fmt.Sprintf(`,"notes":[]`))
    out.WriteString(fmt.Sprintf("}\n"))

    g.WriteFastJSeq(alt_seq[aa], out)
    out.WriteByte('\n')
  }


  out.WriteByte('\n')
  out.Flush()

  return nil
}

func (g *FastJInfo) Process(pasta_stream *bufio.Reader, tag_stream *bufio.Reader, ind int, out *bufio.Writer) error {
  var msg ControlMessage ; _ = msg
  var e error
  var stream0_pos int
  var dbp0 int ; _ = dbp0
  var curStreamState int ; _ = curStreamState

  //out := bufio.NewWriter(os.Stdout)

  bp_count := 0
  lfmod := 50

  for {
    message_processed_flag := false

    ch0,e0 := pasta_stream.ReadByte()
    for (e0==nil) && ((ch0=='\n') || (ch0==' ') || (ch0=='\r') || (ch0=='\t')) {
      ch0,e0 = pasta_stream.ReadByte()
    }
    if e0!=nil { break }

    if ch0=='>' {
      msg,e = process_control_message(pasta_stream)
      if e!=nil { return fmt.Errorf("invalid control message") }

      if (msg.Type == REF) || (msg.Type == NOC) {
        curStreamState = MSG
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

      } else if ind==1 {

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

/*
func (g *RefVarFastJ) Chrom(chr string) { }
func (g *RefVarFastJ) Pos(pos int) { }
func (g *RefVarFastJ) Header(out *bufio.Writer) error { return nil }
func (g *RefVarFastJ) Print(vartype int, ref_start, ref_len int, refseq []byte, altseq [][]byte, out *bufio.Writer) error { return nil  }
func (g *RefVarFastJ) PrintEnd(out *bufio.Writer) error { return nil }

func (g *RefVarFastJ) PastaBegin(out *bufio.Writer) error { return nil }
func (g *RefVarFastJ) Pasta(gvcf_line string, ref_stream *bufio.Reader, out *bufio.Writer) error { return  nil }
func (g *RefVarFastJ) PastaEnd(out *bufio.Writer) error { return nil }
*/
