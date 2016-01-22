package pasta

import "os"
import _ "io"
import "bufio"

import _ "errors"

import "github.com/abeconnelly/simplestream"

var Token []byte
var SubMap map[byte]map[byte]byte
var RefMap map[byte]byte
var AltMap map[byte]byte
var DelMap map[byte]byte
var InsMap map[byte]byte
var IsAltDel map[byte]bool

var RefDelBP map[byte]int


// Ref to Alt
//
//var gSub map[byte]map[byte]byte
var gRefBP map[byte]byte
var gAltBP map[byte]byte
var gPastaBPState map[byte]int


const(
  REF = iota
  SNP = iota
  SUB = iota
  INDEL = iota
  NOC = iota
  FIN = iota
)


func init() {
  Token := []byte("acgtnNACGT~?@=:;#&%*+-QSWd!$7EZ'\",_")

  gPastaBPState = make(map[byte]int)

  DelMap = make(map[byte]byte)
  InsMap = make(map[byte]byte)

  DelMap['a'] = '!'
  DelMap['c'] = '$'
  DelMap['g'] = '7'
  DelMap['t'] = 'E'
  DelMap['n'] = 'z'

  IsAltDel = make(map[byte]bool)
  IsAltDel['!'] = true
  IsAltDel['$'] = true
  IsAltDel['7'] = true
  IsAltDel['E'] = true
  IsAltDel['z'] = true


  InsMap['a'] = 'Q'
  InsMap['c'] = 'S'
  InsMap['g'] = 'W'
  InsMap['t'] = 'd'

  gSub := make(map[byte]map[byte]byte)

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

  // deletion of reference
  //
  gSub['a']['-'] = '!'
  gSub['c']['-'] = '$'
  gSub['t']['-'] = 'E'
  gSub['g']['-'] = '7'
  gSub['n']['-'] = 'z'

  // insertion
  //
  gSub['-'] = make(map[byte]byte)
  gSub['-']['a'] = 'Q'
  gSub['-']['c'] = 'S'
  gSub['-']['g'] = 'W'
  gSub['-']['t'] = 'd'
  gSub['-']['n'] = 'Z'

  gSub['-']['-'] = '.'


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

  //--
  // Alt deleitions

  gRefBP['!'] = 'a'
  gRefBP['$'] = 'c'
  gRefBP['7'] = 'g'
  gRefBP['E'] = 't'
  gRefBP['z'] = 'n'

  // Alt insertions

  gAltBP['Q'] = 'a'
  gAltBP['S'] = 'c'
  gAltBP['W'] = 'g'
  gAltBP['d'] = 't'

  //--

  // no-call substitutions

  gRefBP['\''] = 'n'
  gRefBP['"'] = 'n'
  gRefBP[','] = 'n'
  gRefBP['_'] = 'n'

  gAltBP['\''] = 'a'
  gAltBP['"'] = 'c'
  gAltBP[','] = 'g'
  gAltBP['_'] = 't'

  //-

  gRefBP['n'] = 'n'
  gRefBP['N'] = 'n'

  gAltBP['n'] = 'n'
  gAltBP['N'] = 'n'


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

  SubMap = gSub
  RefMap = gRefBP
  AltMap = gAltBP

  RefDelBP = make(map[byte]int)
  for i:=0; i<len(Token); i++ {
    if _,ok := RefMap[Token[i]] ; ok {
      RefDelBP[Token[i]] = 1
    } else {
      RefDelBP[Token[i]] = 0
    }
  }

}



type PastaHandle struct {
  Fp *os.File
  Scanner *bufio.Scanner

  Stream *simplestream.SimpleStream
  AltStream *simplestream.SimpleStream

  Buf []byte
  Stage []byte
}

/*
func Open(fn string) (p PastaHandle, err error) {
  if fn == "-" {
    p.Fp = os.Stdin
  } else {
    p.Fp,err = os.Open(fn)
  }
  if err!=nil { return }

  p.Reader = bufio.NewReader(p.Fp)
  return p, nil
}

func (p *PastaHandle) Close() {
  p.Fp.Close()
}

func (p *PastaHandle) PeekChar() (byte) {
  if len(p.Stage)==0 { return 0 }
  return p.Stage[0]
}

PASTA_SAUCE := 1024

func (p *PastaHandle) ReadChar() (byte, err) {
  if len(p.Stage)==0 {
    if len(p.Buf)==0 {
      p.Buf = make([]byte, PASTA_SAUCE, PASTA_SAUCE)
    }
    n,e := p.Fp.Read(p.Buf)
    if e!=nil { return 0, e }
    if n==0 { return 0, nil }
    p.Stage = p.Buf[0:n]
  }

  b := p.Stage[0]
  p.Stage = p.Stage[1:]
  return b,nil
}
*/
