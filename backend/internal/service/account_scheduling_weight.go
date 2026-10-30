package service

import (
	"math"
	mathrand "math/rand/v2"
)

// AccountSchedulingWeightExtraKey 账号调度权重（extra 字段，整数）。
// 只在 OpenAI 调度的同一优先级内生效：设了权重（>0）的账号按权重比例随机分流，
// 未设权重（0 或缺省）的账号排在它们之后兜底。同一优先级内没有任何账号设权重时，
// 排序与原来完全一致（负载 → 最久未用 → 组内随机）。
const AccountSchedulingWeightExtraKey = "scheduling_weight"

const maxAccountSchedulingWeight = 1000

// SchedulingWeight 返回账号调度权重，0 表示未设置。
func (a *Account) SchedulingWeight() int {
	if a == nil {
		return 0
	}
	w := a.getExtraInt(AccountSchedulingWeightExtraKey)
	if w < 0 {
		return 0
	}
	if w > maxAccountSchedulingWeight {
		return maxAccountSchedulingWeight
	}
	return w
}

// applySchedulingWeightsWithinPriority 对已按优先级排好序的切片，逐个同优先级段重排：
// 有权重的账号按权重做加权随机排列（Efraimidis–Spirakis：key = u^(1/w)，降序），
// 无权重的账号保持原有相对顺序排在后面。段内没有任何权重时不做改动。
func applySchedulingWeightsWithinPriority[T any](items []T, accountOf func(T) *Account, randFloat func() float64) {
	if len(items) <= 1 {
		return
	}
	if randFloat == nil {
		randFloat = mathrand.Float64
	}
	i := 0
	for i < len(items) {
		priority := accountOf(items[i]).Priority
		j := i + 1
		for j < len(items) && accountOf(items[j]).Priority == priority {
			j++
		}
		reorderTierBySchedulingWeight(items[i:j], accountOf, randFloat)
		i = j
	}
}

func reorderTierBySchedulingWeight[T any](tier []T, accountOf func(T) *Account, randFloat func() float64) {
	type keyed struct {
		item T
		key  float64
	}
	weighted := make([]keyed, 0, len(tier))
	unweighted := make([]T, 0, len(tier))
	for _, item := range tier {
		w := accountOf(item).SchedulingWeight()
		if w <= 0 {
			unweighted = append(unweighted, item)
			continue
		}
		u := randFloat()
		if u <= 0 {
			u = math.SmallestNonzeroFloat64
		}
		weighted = append(weighted, keyed{item: item, key: math.Pow(u, 1/float64(w))})
	}
	if len(weighted) == 0 {
		return
	}
	// 插入排序：同优先级段通常只有几个账号。
	for a := 1; a < len(weighted); a++ {
		for b := a; b > 0 && weighted[b].key > weighted[b-1].key; b-- {
			weighted[b], weighted[b-1] = weighted[b-1], weighted[b]
		}
	}
	n := 0
	for _, k := range weighted {
		tier[n] = k.item
		n++
	}
	for _, item := range unweighted {
		tier[n] = item
		n++
	}
}
