#!/bin/bash

afn="/scratch/l7g/assembly/assembly.00.hg19.fw.gz"
aidx="/scratch/l7g/assembly/assembly.00.hg19.fw.fwi"
tdir="/scratch/l7g/tagset.fa/tagset.fa.gz"


ref="hg19"
chrom="chr5"
path="00fa"

dn=`egrep "$ref:$chrom:$path" $aidx | cut -f2`
st0=`egrep "$ref:$chrom:$path" $aidx | cut -f3`

inpgff="/scratch/pgp/hu826751/hu826751.gff.gz"
reffa="/scratch/ref/hg19.fa/hg19.fa"

en0=`expr $st0 + $dn`

st1=`expr $st0 + 1`
en1=`expr $en0 + 1`

#echo "0ref:" $st0 $en0
#echo "1ref:" $st1 $en1
#echo "gff/ref $chrom:$st1-$en1"


realstart1=`tabix $inpgff $chrom:$st1-$en1 | head -n1 | cut -f4`
realend1=`tabix $inpgff $chrom:$st1-$en1 | tail -n1 | cut -f5`
realdn=`expr $realend1 - $realstart1 + 1`

realstart0=`expr $realstart1 - 1`

# Filter out only information for path 00fa (on chr5)
#
#tabix $inpgff $chrom:$realstart1-$realend1 | \
#  ./pasta -action gff-rotini \
#    -refstream <( samtools faidx $reffa $chrom:$realstart1-$realend1 | egrep -v '^>' | tr '[:upper:]' '[:lower:]'  ) \
#    -start $realstart0 | \
#  ./pasta -action filter-rotini -start $st0 -n $dn | \
#  ./pasta -action rotini-gff
#

tabix $inpgff $chrom:$realstart1-$realend1 | \
  ./pasta -action gff-rotini \
    -refstream <( refstream $reffa $chrom:$realstart1-$realend1 ) \
    -start $realstart0 | \
  ./pasta -action filter-rotini -start $st0 -n $dn | \
  ./pasta -action rotini-gff | \
  ./pasta -action gff-rotini -refstream <( refstream $chrom:$st1-$en1 ) -start $st0 | \
  ./pasta -action rotini-gff | \
  ./pasta -action gff-rotini -refstream <( refstream $chrom:$st1-$en1 ) -start $st0 | \
  ./pasta -action rotini-gff


