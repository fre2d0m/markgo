package mdeditor

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkAnalyze(b *testing.B) {
	const paragraph = "## 中英混合的段落\n\nSome **bold text** with [a link](https://example.com). 中文内容保留源文件。\n\n- [ ] A task\n\n"
	for _, size := range []int{100_000, 1_000_000} {
		source := strings.Repeat(paragraph, size/len(paragraph))
		b.Run(fmt.Sprintf("%dKB", size/1000), func(b *testing.B) {
			b.SetBytes(int64(len(source)))
			b.ReportAllocs()
			for b.Loop() {
				Analyze(source)
			}
		})
	}
}
