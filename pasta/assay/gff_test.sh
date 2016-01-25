#!/bin/bash

odir="assay/gff"
mkdir -p $odir

./pasta -action rstream -param 'p-snp=0.5:ref-seed=11223344:n=1000' > $odir/gff-snp.inp
./pasta -action rotini-gff -i $odir/gff-snp.inp | ./pasta -action gff-rotini -refstream <( ./pasta -action ref-rstream 'ref-seed=11223344' ) > $odir/gff-snp.out

diff <( ./pasta -action rotini-ref -i $odir/gff-snp.inp ) <( ./pasta -action rotini-ref -i $odir/gff-snp.out )
#diff <( ./pasta -action rotini-alt0 -i $odir/gff-snp.inp ) <( ./pasta -aciton rotini-alt0 -i $odir/gff-snp/out )
#diff <( ./pasta -action rotini-alt1 -i $odir/gff-snp.inp ) <( ./pasta -aciton rotini-alt1 -i $odir/gff-snp/out )
