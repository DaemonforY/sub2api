package service

import (
	"fmt"
	"strconv"
	"strings"
)

// siteBuilderRules is what every page must follow; both the first page and later changes use it.
const siteBuilderRules = `你是资深网页设计师兼前端工程师，为用户做一个可以直接上线的单页网站。

输出要求（必须遵守）：
1. 只输出一个完整的 HTML 文件：从 <!DOCTYPE html> 开始，到 </html> 结束。不要 Markdown 代码块，不要任何解释。
2. 所有 CSS 写在 <head> 的一个 <style> 里，需要交互时 JS 写在 </body> 前的一个 <script> 里。
3. 不引用任何外部资源：不用 CDN、Google Fonts、图标库、外链图片和统计脚本（在中国大陆加载不了）。字体用系统字体栈；图标用内联 SVG 或 emoji。
4. 移动端优先、响应式：手机（375px）和电脑（1280px）都要好看。<html lang="zh-CN">，带 viewport meta 和合适的 <title>。
5. 设计要有质感：统一的配色和字号层级、足够的留白、清楚的视觉重点；可以用渐变、圆角、阴影、细腻的动效，但不要花哨。
6. 文案用用户的语言，具体、可信、贴合用户的行业和需求，不要写“Lorem ipsum”。
7. 不要编造真实世界的联系方式、地址、价格、资质、客户评价和人名：用户给了就用用户的，没给就写成明显的占位，例如“电话：请填写”。
8. 网站没有后端：表单不能真的提交。需要“联系我们”时用 mailto:/tel: 链接，或者展示联系方式；如果一定要放表单，提交时用 JS 提示“已记录，请直接联系我们”。
9. 页面总长度控制在 30KB 以内。`

// siteBuilderImageRules tells the model how pictures work: it only writes the description and the
// server draws them.
func siteBuilderImageRules(allowed []int, used map[int]string) string {
	if len(allowed) == 0 && len(used) == 0 {
		return "\n\n图片：这个页面不放 AI 配图，不要写任何 <img>；需要视觉效果时用 CSS 渐变、色块、内联 SVG 图形和 emoji。"
	}
	var b strings.Builder
	_, _ = b.WriteString("\n\n图片（由画图模型另外画，你只写描述）：\n")
	_, _ = b.WriteString(`- 用 <img src="img/编号.jpg" data-ai-prompt="画面描述" data-ai-size="landscape" alt="图片说明"> 放图。data-ai-prompt 写具体的中文画面描述（主体、场景、风格、色调、构图），画面里不要有文字。data-ai-size 取 landscape（横图，默认）、square（方图）或 portrait（竖图）。` + "\n")
	_, _ = b.WriteString("- 图片只能用 <img> 放，不能用 CSS background-image；给 <img> 设好宽高比和 object-fit: cover，图片没加载出来时也不能破坏布局。\n")
	if len(used) > 0 {
		nums := make([]string, 0, len(used))
		for n := 1; n <= siteBuilderMaxImages; n++ {
			if _, ok := used[n]; ok {
				nums = append(nums, strconv.Itoa(n))
			}
		}
		_, _ = b.WriteString("- 页面里已有的图片编号：" + strings.Join(nums, "、") + "。不改的图片原样保留它的 src 和 data-ai-prompt；要换画面就改 data-ai-prompt（会重新画）；删掉的图片直接去掉 <img>。\n")
	}
	if len(allowed) > 0 {
		nums := make([]string, len(allowed))
		for i, n := range allowed {
			nums[i] = strconv.Itoa(n)
		}
		_, _ = b.WriteString("- 可以新增图片的编号：" + strings.Join(nums, "、") + "（不需要就不用）。每个编号只用一次，不要用其他编号。\n")
	} else {
		_, _ = b.WriteString("- 不能再新增图片。\n")
	}
	return b.String()
}

func siteBuilderCreatePrompt(images int) string {
	allowed := make([]int, 0, images)
	for n := 1; n <= images; n++ {
		allowed = append(allowed, n)
	}
	extra := ""
	if images > 0 {
		extra = fmt.Sprintf("\n- 这次请用满 %d 张图，第 1 张作为首屏主视觉。", images)
	}
	return siteBuilderRules + siteBuilderImageRules(allowed, nil) + extra
}

func siteBuilderCreateRequest(b SiteDraftBrief) string {
	var s strings.Builder
	_, _ = s.WriteString("网站需求：\n" + b.Description)
	if b.Style != "" {
		_, _ = s.WriteString("\n\n风格偏好：" + b.Style)
	}
	_, _ = s.WriteString("\n\n请直接输出完整的 HTML。")
	return s.String()
}

func siteBuilderRevisePrompt(allowed []int, used map[int]string) string {
	return siteBuilderRules + siteBuilderImageRules(allowed, used) + `

现在用户要修改已有的页面：
- 按用户的要求改，没提到的部分保持原样（包括文案、配色和结构），不要借机重新设计。
- 输出修改后的完整 HTML（不是差异）。`
}

func siteBuilderReviseRequest(html, instruction string) string {
	return "当前页面：\n" + html + "\n\n用户的修改要求：\n" + instruction + "\n\n请输出修改后的完整 HTML。"
}
