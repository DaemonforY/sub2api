<script setup lang="ts">
// Student perks on the campus page, read from the live settings (free calls, education discount,
// invitee bonus) so the page never promises more than the site gives.
import { onMounted, ref } from 'vue'
import { growthConfig, learnConfig, type GrowthConfig, type LearnConfig } from '../api'
import { MAIN_SITE } from '../tracks'

const learn = ref<LearnConfig | null>(null)
const growth = ref<GrowthConfig | null>(null)

onMounted(async () => {
  ;[learn.value, growth.value] = await Promise.all([learnConfig(), growthConfig()])
})

function zhe(percent: number) {
  const v = (100 - percent) / 10
  return Number.isInteger(v) ? String(v) : v.toFixed(1)
}
</script>

<template>
  <div class="campus-perks" data-testid="campus-perks">
    <div class="campus-perk">
      <span class="campus-perk-icon">🧪</span>
      <strong>课程免费学，代码在线跑</strong>
      <p v-if="learn && learn.run_enabled && learn.own_key_only">
        全部课程免费阅读；登录后用自己的 HiveGPT Key 在页面里运行代码、问 AI 助教、做模拟面试，按实际用量计费。
      </p>
      <p v-else-if="learn && learn.run_enabled">
        登录后每天 {{ learn.free_runs_per_day }} 次免费运行、{{ learn.tutor_free_per_day }} 次问 AI 助教、{{ learn.interviews_per_day }} 场模拟面试，不用先充值。
      </p>
      <p v-else>全部课程免费阅读，代码都可以复制到本地用自己的 Key 运行。</p>
    </div>
    <div v-if="growth && growth.edu_verify_enabled" class="campus-perk">
      <span class="campus-perk-icon">🎓</span>
      <strong>
        学校邮箱认证<template v-if="growth.edu_discount_percent > 0">，订阅 {{ zhe(growth.edu_discount_percent) }} 折</template>
      </strong>
      <p>
        用 {{ growth.edu_email_suffixes.join('、') }} 结尾的学校邮箱验证一次（注册邮箱不是学校邮箱也可以），<template v-if="growth.edu_discount_percent > 0">购买订阅立减 {{ growth.edu_discount_percent }}%，</template>开启了教育价的付费课程也按教育价结算。
      </p>
      <a class="runbox-btn small" :href="`${MAIN_SITE}/profile`">去认证</a>
    </div>
    <div class="campus-perk">
      <span class="campus-perk-icon">📜</span>
      <strong>结业证书写进简历</strong>
      <p>每张证书都有公开的验证链接，附上你的结业项目。面试官点开就能看到你学了什么、做了什么。</p>
    </div>
    <div v-if="growth && growth.affiliate_enabled" class="campus-perk">
      <span class="campus-perk-icon">🤝</span>
      <strong>拉上同学一起学</strong>
      <p>
        把你的邀请链接发给同学。<template v-if="growth.invitee_bonus_rate_percent > 0">同学通过链接注册，首笔付费订单额外返 {{ growth.invitee_bonus_rate_percent }}% 到余额；</template>你也能在「邀请返利」里拿到返利。
      </p>
      <a class="runbox-btn small ghost" :href="`${MAIN_SITE}/affiliate`">我的邀请链接</a>
    </div>
  </div>
</template>
