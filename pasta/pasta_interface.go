package main

import "bufio"

import "github.com/abeconnelly/simplestream"

type RefVarPrinter interface {
  Header(out *bufio.Writer) error
  Print(vartype int, ref_start, ref_len int, refseq []byte, altseq [][]byte, out *bufio.Writer) error
  Pasta(line string, ref_stream *simplestream.SimpleStream, out *bufio.Writer) error
  PastaBegin(out *bufio.Writer) error
  PastaEnd(out *bufio.Writer) error
  Chrom(chr string)
  Pos(pos int)
  Init()
}
