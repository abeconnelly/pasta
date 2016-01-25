package main

type RefVarPrinter interface {
  Header() string
  Print(vartype int, ref_start, ref_len int, refseq []byte, altseq [][]byte) error
  Chrom(chr string)
  Pos(pos int)
  Init()
}
