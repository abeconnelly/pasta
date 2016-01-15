package main

import "math/rand"
import "fmt"
import "strconv"
import "strings"

import "github.com/abeconnelly/pasta"


type RandomStreamContext struct {
  Allele          int
  N               int
  Seed            int64

  RefLen          []int

  PNocallLocked   float64
  PNocall         float64
  NocallLen       []int

  PSnpLocked      float64
  PSnp            float64

  PIndel          float64
  PIndelLocked    float64
  IndelLen        []int

  Chrom           string
  Pos             uint64

  LFMod           int

  Rnd             *rand.Rand
}

func default_random_stream_context() *RandomStreamContext {
  ctx := RandomStreamContext{}
  ctx.Allele = 2
  ctx.N = 1000
  ctx.Seed = 0xabecafe

  ctx.RefLen = []int{1, 50}

  ctx.PNocallLocked = 0.95
  ctx.PNocall = 10.0/float64(ctx.N)
  ctx.NocallLen = []int{1, 10}

  ctx.PSnpLocked = 0.7
  ctx.PSnp = 1.0/200.0

  ctx.PIndel = 1.0/1000.0
  ctx.PIndelLocked = 0.125
  ctx.IndelLen = []int{-10,10}

  ctx.Chrom = "Unk"
  ctx.Pos = 0

  ctx.LFMod = 50

  src := rand.NewSource(ctx.Seed)
  rnd := rand.New(src)

  ctx.Rnd = rnd

  return &ctx
}

func parsei(val string, def_val int) int {
  z,e := strconv.ParseInt(val, 10, 64)
  if e!=nil { return def_val }
  return int(z)
}

func parsei64(val string, def_val int64) int64 {
  z,e := strconv.ParseInt(val, 10, 64)
  if e!=nil { return def_val }
  return z
}

func parseui64(val string, def_val uint64) uint64 {
  z,e := strconv.ParseUint(val, 10, 64)
  if e!=nil { return def_val }
  return z
}

func parseu(val string, def_val uint) uint {
  z,e := strconv.ParseUint(val, 10, 64)
  if e!=nil { return def_val }
  return uint(z)
}

func parsef(val string, def_val float64) float64 {
  z,e := strconv.ParseFloat(val, 64)
  if e!=nil { return def_val }
  return float64(z)
}

func random_stream_context_from_param(param string) *RandomStreamContext {
  ctx := default_random_stream_context()

  orig_seed := ctx.Seed

  if param=="" { return ctx }

  param_parts := strings.Split(param, ":")
  for i:=0; i<len(param_parts); i++ {
    val_parts := strings.Split(param_parts[i], "=")
    if len(val_parts)!=2 { continue }

    if val_parts[0] == "allele" {
      ctx.Allele = parsei(val_parts[1], ctx.Allele)
    } else if val_parts[0] == "n" {
      ctx.N = parsei(val_parts[1], ctx.N)
    } else if val_parts[0] == "seed" {
      ctx.Seed = parsei64(val_parts[1], ctx.Seed)
    } else if val_parts[0] == "pos" {
      ctx.Pos = parseui64(val_parts[1], ctx.Pos)
    } else if val_parts[0] == "chrom" {
      ctx.Chrom = val_parts[1]
    } else if val_parts[0] == "lfmod" {
      ctx.LFMod = parsei(val_parts[1], ctx.LFMod)

    } else if val_parts[0] == "p-nocall-locked" {
      ctx.PNocallLocked = parsef(val_parts[1], ctx.PNocallLocked)
    } else if val_parts[0] == "p-nocall" {
      ctx.PNocall = parsef(val_parts[1], ctx.PNocall)
    } else if val_parts[0] == "p-nocall-length" {
      l_parts := strings.Split(val_parts[1], ",")
      L := len(l_parts)
      if L >= 2 { L = 2 }
      for ii:=0; ii<L; ii++ {
        ctx.NocallLen[ii] = parsei(l_parts[ii], ctx.NocallLen[ii])
      }

    } else if val_parts[0] == "p-snp-locked" {
      ctx.PSnpLocked = parsef(val_parts[1], ctx.PSnpLocked)
    } else if val_parts[0] == "p-snp" {
      ctx.PSnp = parsef(val_parts[1], ctx.PSnp)

    } else if val_parts[0] == "p-indel-locked" {
      ctx.PIndelLocked = parsef(val_parts[1], ctx.PIndelLocked)
    } else if val_parts[0] == "p-indel" {
      ctx.PIndel = parsef(val_parts[1], ctx.PIndel)
    } else if val_parts[0] == "p-indel-length" {
      l_parts := strings.Split(val_parts[1], ",")
      L := len(l_parts)
      if L >= 2 { L = 2 }
      for ii:=0; ii<L; ii++ {
        ctx.IndelLen[ii] = parsei(l_parts[ii], ctx.IndelLen[ii])
      }


    } else if val_parts[0] == "p-indel-locked" {
      ctx.PIndelLocked = parsef(val_parts[1], ctx.PIndelLocked)
    } else if val_parts[0] == "p-indel" {
      ctx.PIndel = parsef(val_parts[1], ctx.PIndel)
    } else if val_parts[0] == "p-indel-length" {
      l_parts := strings.Split(val_parts[1], ",")
      L := len(l_parts)
      if L >= 2 { L = 2 }
      for ii:=0; ii<L; ii++ {
        ctx.IndelLen[ii] = parsei(l_parts[ii], ctx.IndelLen[ii])
      }


    }

  }

  if ctx.Seed != orig_seed {
    src := rand.NewSource(ctx.Seed)
    rnd := rand.New(src)
    ctx.Rnd = rnd
  }

  return ctx

}

func random_state_pick(ctx *RandomStreamContext) (int,[]int) {

  rnd := ctx.Rnd

  _z := []int{}


  p := rnd.Float64()
  if p < ctx.PNocall {
    _z = append(_z, rnd.Intn(ctx.NocallLen[1] - ctx.NocallLen[0]) + ctx.NocallLen[0])
    p = rnd.Float64()
    if p >= ctx.PNocallLocked {
      for a:=0; a<ctx.Allele; a++ {
        _z = append(_z, rnd.Intn(ctx.NocallLen[1] - ctx.NocallLen[0]) + ctx.NocallLen[0])
      }
    } else {
      for a:=0; a<ctx.Allele; a++ {
        _z = append(_z, _z[0])
      }
    }
    return NOC, _z
  }

  p = rnd.Float64()
  if p < ctx.PSnp {
    p = rnd.Float64()
    _z = append(_z, rnd.Intn(4))
    if p >= ctx.PSnpLocked {
      for a:=0; a<ctx.Allele; a++ {
        _z = append(_z, rnd.Intn(4))
      }
    } else {
      for a:=0; a<ctx.Allele; a++ {
        _z = append(_z, _z[0])
      }
    }
    return SNP, _z
  }

  p = rnd.Float64()
  if p < ctx.PIndel {
    _z = append(_z, rnd.Intn(ctx.IndelLen[1] - ctx.IndelLen[0]) + ctx.IndelLen[0])
    p = rnd.Float64()
    if p >= ctx.PIndelLocked {
      for a:=0; a<ctx.Allele; a++ {
        _z = append(_z, rnd.Intn(ctx.IndelLen[1] - ctx.IndelLen[0]) + ctx.IndelLen[0])
      }
    } else {
      for a:=0; a<ctx.Allele; a++ {
        _z = append(_z, _z[0])
      }
    }
    return INDEL, _z
  }

  _z = append(_z, rnd.Intn(ctx.RefLen[1] - ctx.RefLen[0]) + ctx.RefLen[0])
  return REF, _z
}

func random_ref_bp(ctx *RandomStreamContext) byte {
  rnd := ctx.Rnd

  var ref_bp byte
  ref_bp_i := rnd.Intn(4)

  if ref_bp_i == 0 {
    ref_bp = 'a'
  } else if ref_bp_i == 1 {
    ref_bp = 'c'
  } else if ref_bp_i == 2 {
    ref_bp = 't'
  } else if ref_bp_i == 3 {
    ref_bp = 'g'
  }

  return ref_bp
}

func _ibp(_i int) byte {
  if _i == 0 {
     return 'a'
  } else if _i == 1 {
     return 'c'
  } else if _i == 2 {
     return 't'
  } else if _i == 3 {
     return 'g'
  }
  return '-'
}

//func random_stream(out io.Writer, ctx *RandomStreamContext) {
func random_stream(ctx *RandomStreamContext) {
  if ctx==nil {
    ctx = default_random_stream_context()
  }

  //src := rand.NewSource(ctx.Seed)
  //rnd := rand.New(src)

  fmt.Printf(">C{%s}>P{%d}>#{random_stream}\n", ctx.Chrom, ctx.Pos)

  for bp_count:=0; bp_count<ctx.N; {

    state,lparts := random_state_pick(ctx)

    for a:=0; a<len(lparts); a++ {
      if bp_count+lparts[a] > ctx.N { lparts[a] = ctx.N-bp_count }
    }

    if state==REF {

      for ii:=0; ii<lparts[0]; ii++ {

        ref_bp := random_ref_bp(ctx)
        for a:=0; a<ctx.Allele; a++ {
          fmt.Printf("%c", ref_bp)

          bp_count++
          if (ctx.LFMod>0) && ((bp_count%ctx.LFMod)==0) {
            fmt.Printf("\n")
          }
        }

      }

      continue

    } else if state==SNP {

      ref_bp := random_ref_bp(ctx)

      for a:=0; a<ctx.Allele; a++ {
        snp := _ibp(lparts[a])

        if ref_bp == snp {
          fmt.Printf("%c", ref_bp)
        } else {
          fmt.Printf("%c", pasta.SubMap[ref_bp][snp])
        }

        bp_count++
        if (ctx.LFMod>0) && ((bp_count%ctx.LFMod)==0) {
          fmt.Printf("\n")
        }

      }
    }

  }

  fmt.Printf("\n")

}
