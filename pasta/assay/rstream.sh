#!/bin/bash

mkdir -p assay

function _q {
  echo $1
  exit 1
}

# diploid stream
#
./pasta -action rstream -F -param 'allele=2:n=10000:seed=1234:p-snp=0.3:p-snp-locked=0.5:seed=1234' > assay/a.0
./pasta -action rotini-diff -i assay/a.0 -F | ./pasta -action diff-rotini -i - > assay/b.0
diff assay/a.0 assay/b.0 || ( echo "mismatch 0" && exit 1 )

# two diploid streams concatenated
#
cat <( ./pasta -action rstream -param 'pos=0:n=100:seed=1234' ) <( ./pasta -action rstream -param 'pos=200:chrom=chr1:n=100:seed=4321' ) | sed '/^$/d' > assay/a.1
./pasta -action rotini-diff -i assay/a.1 -F | ./pasta -action diff-rotini -i - | sed '/^$/d' > assay/b.1
diff assay/a.1 assay/b.1 || ( echo "mismatch 1" && exit 1 )

# nocall (locked)
#
./pasta -action rstream -F -param 'allele=2:n=10000:seed=1234:p-nocall=0.3:p-nocall-locked=1.0:seed=1234' > assay/a.2
./pasta -action rotini-diff -i assay/a.2 -F | ./pasta -action diff-rotini -i - > assay/b.2
diff assay/a.2 assay/b.2 || ( echo "mismatch 2" && exit 1 )


# test indels
#
ofn_b="assay/indel"
./pasta -action rstream -param 'p-indel=0.5:p-indel-locked=0.8:p-indel-length=0,3:seed=1234' > $ofn_b.inp
./pasta -action rotini-ref -i $ofn_b.inp > $ofn_b.inp.ref
./pasta -action rotini-alt0 -i $ofn_b.inp > $ofn_b.inp.alt0
./pasta -action rotini-alt1 -i $ofn_b.inp > $ofn_b.inp.alt1


./pasta -action rotini-diff -i $ofn_b.inp -F | ./pasta -action diff-rotini > $ofn_b.out
./pasta -action rotini-ref -i $ofn_b.out > $ofn_b.out.ref
./pasta -action rotini-alt0 -i $ofn_b.out > $ofn_b.out.alt0
./pasta -action rotini-alt1 -i $ofn_b.out > $ofn_b.out.alt1

#diff <( ./pasta -action rotini-diff -i $ofn_b.inp -F ) <( ./pasta -action rotini-diff -i $ofn_b.out -F ) || ( echo "indel diff mismatch" && exit 1 )
diff $ofn_b.inp.ref $ofn_b.out.ref || ( echo "indel ref mismatch" && exit 1 )
diff $ofn_b.inp.alt0 $ofn_b.out.alt0 || ( echo "indel alt0 mismatch" && exit 1 )
diff $ofn_b.inp.alt1 $ofn_b.out.alt1 || ( echo "indel alt1 mismatch" && exit 1 )


## snp and nocall
#
ofn_b="assay/snp_nocall"
./pasta -action rstream -param 'p-snp=0.8:p-snp-nocall=0.5:seed=1234:p-snp-locked=0.0' > $ofn_b.inp

./pasta -action rotini-ref -i $ofn_b.inp > $ofn_b.inp.ref
./pasta -action rotini-alt0 -i $ofn_b.inp > $ofn_b.inp.alt0
./pasta -action rotini-alt1 -i $ofn_b.inp > $ofn_b.inp.alt1

./pasta -action rotini-diff -i $ofn_b.inp -F --full-nocall-sequence | ./pasta -action diff-rotini > $ofn_b.out
./pasta -action rotini-ref -i $ofn_b.out > $ofn_b.out.ref
./pasta -action rotini-alt0 -i $ofn_b.out > $ofn_b.out.alt0
./pasta -action rotini-alt1 -i $ofn_b.out > $ofn_b.out.alt1

#diff <( ./pasta -action rotini-diff -i $ofn_b.inp -F ) <( ./pasta -action rotini-diff -i $ofn_b.out -F ) || ( echo "indel diff mismatch" && exit 1 )
diff $ofn_b.inp.ref $ofn_b.out.ref || ( echo "indel ref mismatch" && exit 1 )
diff $ofn_b.inp.alt0 $ofn_b.out.alt0 || ( echo "indel alt0 mismatch" && exit 1 )
diff $ofn_b.inp.alt1 $ofn_b.out.alt1 || ( echo "indel alt1 mismatch" && exit 1 )


## indel and nocall
#
ofn_b="assay/indel_nocall"
./pasta -action rstream -param 'p-indel=0.5:p-indel-nocall=0.5:seed=1234:n=5000' > $ofn_b.inp

./pasta -action rotini-ref -i $ofn_b.inp > $ofn_b.inp.ref
./pasta -action rotini-alt0 -i $ofn_b.inp > $ofn_b.inp.alt0
./pasta -action rotini-alt1 -i $ofn_b.inp > $ofn_b.inp.alt1

./pasta -action rotini-diff -i $ofn_b.inp -F --full-nocall-sequence | ./pasta -action diff-rotini > $ofn_b.out
./pasta -action rotini-ref -i $ofn_b.out > $ofn_b.out.ref
./pasta -action rotini-alt0 -i $ofn_b.out > $ofn_b.out.alt0
./pasta -action rotini-alt1 -i $ofn_b.out > $ofn_b.out.alt1

#diff <( ./pasta -action rotini-diff -i $ofn_b.inp -F --full-nocall-sequence ) <( ./pasta -action rotini-diff -i $ofn_b.out -F --full-nocall-sequence) ||_q "indel diff mismatch"
diff $ofn_b.inp.ref $ofn_b.out.ref || _q "indel ref mismatch"
diff $ofn_b.inp.alt0 $ofn_b.out.alt0 || _q  "indel alt0 mismatch"
diff $ofn_b.inp.alt1 $ofn_b.out.alt1 || _q "indel alt1 mismatch"



echo ok
exit 0
