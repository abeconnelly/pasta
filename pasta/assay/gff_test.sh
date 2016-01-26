#!/bin/bash

odir="assay/gff"
mkdir -p $odir

## GFF with snps
##
./pasta -action rstream -param 'p-snp=0.5:ref-seed=11223344:n=1000:seed=1234' > $odir/gff-snp.inp
./pasta -action rotini-gff -i $odir/gff-snp.inp | ./pasta -action gff-rotini -refstream <( ./pasta -action ref-rstream -param 'ref-seed=11223344:allele=1' ) > $odir/gff-snp.out

diff <( ./pasta -action rotini-ref -i $odir/gff-snp.inp ) <( ./pasta -action rotini-ref -i $odir/gff-snp.out )
diff <( ./pasta -action rotini-alt0 -i $odir/gff-snp.inp ) <( ./pasta -action rotini-alt0 -i $odir/gff-snp.out )
diff <( ./pasta -action rotini-alt1 -i $odir/gff-snp.inp ) <( ./pasta -action rotini-alt1 -i $odir/gff-snp.out )

## GFF with indels
##
./pasta -action rstream -param 'p-indel=0.5:p-indel-length=0,3:ref-seed=11223344:n=1000:seed=1234' > $odir/gff-indel.inp
./pasta -action rotini-gff -i $odir/gff-indel.inp | ./pasta -action gff-rotini -refstream <( ./pasta -action ref-rstream -param 'ref-seed=11223344:allele=1' ) > $odir/gff-indel.out

echo ref
diff <( ./pasta -action rotini-ref -i $odir/gff-indel.inp ) <( ./pasta -action rotini-ref -i $odir/gff-indel.out )
echo

echo alt0
diff <( ./pasta -action rotini-alt0 -i $odir/gff-indel.inp ) <( ./pasta -action rotini-alt0 -i $odir/gff-indel.out )
echo

echo alt1
diff <( ./pasta -action rotini-alt1 -i $odir/gff-indel.inp ) <( ./pasta -action rotini-alt1 -i $odir/gff-indel.out )
echo




echo ok
exit 0
