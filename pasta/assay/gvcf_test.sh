#!/bin/bash

odir="assay/gvcf"
mkdir -p $odir

## GVCF with snps
##
./pasta -action rstream -param 'p-snp=0.5:ref-seed=11223344:n=1000:seed=1234' > $odir/gvcf-snp.inp
./pasta -action rotini-gvcf -i $odir/gvcf-snp.inp | ./pasta -action gvcf-rotini -refstream <( ./pasta -action ref-rstream -param 'ref-seed=11223344:allele=1' ) > $odir/gvcf-snp.out

diff <( ./pasta -action rotini-ref -i $odir/gvcf-snp.inp ) <( ./pasta -action rotini-ref -i $odir/gvcf-snp.out )
diff <( ./pasta -action rotini-alt0 -i $odir/gvcf-snp.inp ) <( ./pasta -action rotini-alt0 -i $odir/gvcf-snp.out )
diff <( ./pasta -action rotini-alt1 -i $odir/gvcf-snp.inp ) <( ./pasta -action rotini-alt1 -i $odir/gvcf-snp.out )

echo ok-snp

## GVCF with indels
##
./pasta -action rstream -param 'p-indel=0.8:p-indel-length=0,3:p-nocall=0:ref-seed=11223344:n=1000:seed=1234' > $odir/gvcf-indel.inp
./pasta -action rotini-gvcf -i $odir/gvcf-indel.inp | ./pasta -action gvcf-rotini -refstream <( ./pasta -action ref-rstream -param 'ref-seed=11223344:allele=1' ) > $odir/gvcf-indel.out

diff <( ./pasta -action rotini-ref -i $odir/gvcf-indel.inp ) <( ./pasta -action rotini-ref -i $odir/gvcf-indel.out )
diff <( ./pasta -action rotini-alt0 -i $odir/gvcf-indel.inp ) <( ./pasta -action rotini-alt0 -i $odir/gvcf-indel.out )
diff <( ./pasta -action rotini-alt1 -i $odir/gvcf-indel.inp ) <( ./pasta -action rotini-alt1 -i $odir/gvcf-indel.out )


echo ok-indel

## GVCF with nocall
##
ofn_b="gvcf-nocall"
./pasta -action rstream -param 'p-nocall=0.3:ref-seed=11223344:seed=1234' > $odir/$ofn_b.inp
./pasta -action rotini-gvcf -i $odir/$ofn_b.inp | ./pasta -action gvcf-rotini -refstream <( ./pasta -action ref-rstream -param 'ref-seed=11223344:allele=1' )  > $odir/$ofn_b.out


diff <( ./pasta -action rotini-ref -i $odir/$ofn_b.inp ) <( ./pasta -action rotini-ref -i $odir/$ofn_b.out )
diff <( ./pasta -action rotini-alt0 -i $odir/$ofn_b.inp ) <( ./pasta -action rotini-alt0 -i $odir/$ofn_b.out )
diff <( ./pasta -action rotini-alt1 -i $odir/$ofn_b.inp ) <( ./pasta -action rotini-alt1 -i $odir/$ofn_b.out )

echo ok-nocall


## GVCF with indels and nocalls
##
#./pasta -action rstream -param 'p-nocall=0.3:p-indel=0.3:ref-seed=11223344:seed=1234'  > $odir/gvcf-indel-nocall.inp
./pasta -action rstream -param 'p-nocall=0.3:p-indel=0.3:p-indel-nocall=0.8:ref-seed=11223344:seed=1234'  > $odir/gvcf-indel-nocall.inp
./pasta -action rotini-gvcf -i $odir/gvcf-indel-nocall.inp | ./pasta -action gvcf-rotini -refstream <( ./pasta -action ref-rstream -param 'ref-seed=11223344:allele=1' ) > $odir/gvcf-indel-nocall.out

diff <( ./pasta -action rotini-ref -i $odir/gvcf-indel-nocall.inp ) <( ./pasta -action rotini-ref -i $odir/gvcf-indel-nocall.out ) || echo "error ref"
diff <( ./pasta -action rotini-alt0 -i $odir/gvcf-indel-nocall.inp ) <( ./pasta -action rotini-alt0 -i $odir/gvcf-indel-nocall.out ) || echo "error alt0"
diff <( ./pasta -action rotini-alt1 -i $odir/gvcf-indel-nocall.inp ) <( ./pasta -action rotini-alt1 -i $odir/gvcf-indel-nocall.out ) || echo "error alt1"

echo ok-indel-nocall


exit 0

#diff $odir/gvcf-nocall.inp $odir/gvcf-nocall.out
#diff <( cat $odir/gvcf-nocall.inp | tr -d '\n' | fold -w 50 ) <( cat $odir/gvcf-nocall.out | tr -d '\n'  | fold -w 50 )
diff <( cat $odir/gvcf-nocall.inp | tr -d '\n' | sed 's/[ACTG]*$//' | fold -w 50 ) <( cat $odir/gvcf-nocall.out | tr -d '\n' | sed 's/[ACTG]*$//' | fold -w 50 )


## GVCF with het nocall
##
refseed="11223344"
altseed="1234"

param_inp="p-indel-nocall=0.5:p-indel=0.5:ref-seed=$refseed:seed=$altseed:p-nocall=0.3"
param_ref="ref-seed=$refseed:allele=1"

./pasta -action rstream -param "$param_inp" > $odir/gvcf-indel-nocall.inp
./pasta -action rotini-gvcf -i $odir/gvcf-indel-nocall.inp | ./pasta -action gvcf-rotini -refstream <( ./pasta -action ref-rstream -param "$param_ref" ) > $odir/gvcf-indel-nocall.out

diff <( ./pasta -action rotini-ref -i $odir/gvcf-indel-nocall.inp ) <( ./pasta -action rotini-ref -i $odir/gvcf-indel-nocall.out )
diff <( ./pasta -action rotini-alt0 -i $odir/gvcf-indel-nocall.inp ) <( ./pasta -action rotini-alt0 -i $odir/gvcf-indel-nocall.out )
diff <( ./pasta -action rotini-alt1 -i $odir/gvcf-indel-nocall.inp ) <( ./pasta -action rotini-alt1 -i $odir/gvcf-indel-nocall.out )

echo ok
exit 0
