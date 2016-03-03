#!/bin/bash



afn="/scratch/l7g/assembly/assembly.00.hg19.fw.gz"
aidx="/scratch/l7g/assembly/assembly.00.hg19.fw.fwi"
tdir="/scratch/l7g/tagset.fa/tagset.fa.gz"


ref="hg19"
chrom="chr5"
path="00fa"

#dn=`egrep "$ref:$chrom:$path" $aidx | cut -f2`
#st0=`egrep "$ref:$chrom:$path" $aidx | cut -f3`

inpgff="/scratch/pgp/hu826751/hu826751.gff.gz"
reffa="/scratch/ref/hg19.fa/hg19.fa"

#en0=`expr $st0 + $dn`
#st1=`expr $st0 + 1`
#en1=`expr $en0 + 1`

ucpath=`echo $path | tr '[:lower:]' '[:upper:]'`
prevpath=`echo -e "ibase=16\n$ucpath - 1" | bc -q  | tr '[:upper:]' '[:lower:]'`
prevpath=`printf "%04x" $prevpath`

st0=`l7g assembly $afn $prevpath | tail -n1 | cut -f2`
en0=`l7g assembly $afn $path | tail -n1 | cut -f2`
dn=`expr $en0 - $st0`

st1=`expr $st0 + 1`
en1=`expr $en0 + 1`


realstart1=`tabix $inpgff $chrom:$st1-$en1 | head -n1 | cut -f4`
realend1=`tabix $inpgff $chrom:$st1-$en1 | tail -n1 | cut -f5`
realdn=`expr $realend1 - $realstart1 + 1`

realstart0=`expr $realstart1 - 1`

#ambly_line="hg19:$chrom:$path"
#ambly_beg=`egrep "$ambly_line" $aidx | cut -f3`
#ambly_len=`egrep "$ambly_line" $aidx | cut -f2`
#ambly_end=`expr "$ambly_beg" + "$ambly_len"`
#
#prev_ambly_line="hg19:$chrom:$prevpath"
#prev_ambly_beg=`egrep "$prev_ambly_line" $aidx | cut -f3`
#prev_ambly_len=`egrep "$prev_ambly_line" $aidx | cut -f2`
#prev_ambly_end=`expr "$prev_ambly_beg" + "$prev_ambly_len"`

#x=`expr $prev_ambly_beg + $prev_ambly_len - 15`

#path_start=`bgzip -c -b $x -s 15 $afn | cut -f2`
#echo path_start $path_start
#echo $ucpath $prevpath

echo tabix $inpgff $chrom:$realstart1-$realend1

echo  ./pasta -action gff-rotini \
    -refstream \<\( refstream $reffa $chrom:$realstart1-$realend1 \) \
    -start $realstart0

echo ./pasta -action filter-rotini -start $st0 -n $dn

echo ./pasta -action rotini-fastj -start $st0 \
    -assembly \<\( bgzip -c -b $ambly_beg -s $ambly_len $afn \) \
    -tag \<\( samtools faidx $tdir $path.00 "| egrep -v '^>' | tr -d '\n' | fold -w 24" \)

#tabix $inpgff $chrom:$realstart1-$realend1 | \
#  ./pasta -action gff-rotini \
#    -refstream <( refstream $reffa $chrom:$realstart1-$realend1 ) \
#    -start $realstart0 | \
#  ./pasta -action filter-rotini -start $st0 -n $dn | \
#  egrep -v '^>'

tabix $inpgff $chrom:$realstart1-$realend1 | \
  ./pasta -action gff-rotini \
    -refstream <( refstream $reffa $chrom:$realstart1-$realend1 ) \
    -start $realstart0 | \
  ./pasta -action filter-rotini -start $st0 -n $dn | \
  egrep -v '^>' | \
  ./pasta -action rotini-fastj -start $st0 -tilepath $path -chrom $chrom -build $ref \
  -assembly <( l7g assembly $afn $path ) \
    -tag <( samtools faidx $tdir $path.00 | egrep -v '^>' | tr -d '\n' | fold -w 24 )
