// 端到端验证：直接调 ClaudeAdapter 跑真数据。
// 用法：go run ./cmd/verify
//
// 跑通后看输出，能确认：
//   - DetectProcess 找到真 Claude Code 进程
//   - 19 项能力的 parser 都工作
//   - 真 session 文件被正确解析
//   - 真实 USD 计算

package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/narcilee7/aland/backend/tribes"
)

func main() {
	fmt.Println("=== Aland 端到端验证 · 真数据 ===")
	fmt.Println()

	adp := tribes.NewClaudeAdapter("")
	caps := adp.Capabilities()

	fmt.Printf("Capabilities 声明：\n")
	fmt.Printf("  process=%v launch=%v config=%v configEdit=%v\n", caps.Process, caps.Launch, caps.Config, caps.ConfigEdit)
	fmt.Printf("  sessions=%v sessionTail=%v tokens=%v tokensLive=%v\n", caps.Sessions, caps.SessionTail, caps.Tokens, caps.TokensLive)
	fmt.Printf("  features: %d 个\n", len(caps.Features))
	for _, f := range caps.Features {
		fmt.Printf("    - %s: %s (hasData=%v)\n", f.ID, f.Label, f.HasData)
	}
	fmt.Println()

	// 1. DetectProcess
	fmt.Println("--- 1. DetectProcess ---")
	if proc, err := adp.DetectProcess(); err != nil {
		fmt.Printf("  ❌ 错误: %v\n", err)
	} else if proc == nil {
		fmt.Println("  ⚠️  没有 Claude Code 进程在跑")
	} else {
		fmt.Printf("  ✓ 找到 Claude Code 进程\n")
		fmt.Printf("    PID:        %d\n", proc.PID)
		fmt.Printf("    CPU:        %.1f%%\n", proc.CPU)
		fmt.Printf("    Memory:     %.0f MB\n", proc.Memory)
		fmt.Printf("    CWD:        %s\n", proc.CWD)
		fmt.Printf("    Started:    %s\n", time.Unix(proc.StartTime, 0).Format("15:04:05"))
		fmt.Printf("    CmdLine:    %s\n", proc.CmdLine)
	}
	fmt.Println()

	// 2. ParseConfigDNA
	fmt.Println("--- 2. ParseConfigDNA ---")
	if dna, err := adp.ParseConfigDNA(); err != nil {
		fmt.Printf("  ❌ 错误: %v\n", err)
	} else {
		fmt.Printf("  ✓ source: %s\n", dna.Source)
		fmt.Printf("  Surface: %d 项, Middle: %d 项, Deep: %d 项\n", len(dna.Surface), len(dna.Middle), len(dna.Deep))
		fmt.Printf("  Schema:  %d field meta\n", len(dna.Schema.Fields))
		if len(dna.Surface) > 0 {
			fmt.Printf("    Surface[0]: %s = %v\n", dna.Surface[0].Key, dna.Surface[0].Value)
		}
		if len(dna.Middle) > 0 {
			fmt.Printf("    Middle[0]:  %s = %s (sensitive=%v)\n", dna.Middle[0].Key, mask(dna.Middle[0].Value), dna.Middle[0].Sensitive)
		}
	}
	fmt.Println()

	// 3. MCPServers
	fmt.Println("--- 3. ListMCPServers ---")
	if mcps, err := adp.ListMCPServers(); err != nil {
		fmt.Printf("  ❌ 错误: %v\n", err)
	} else {
		fmt.Printf("  ✓ %d 个 MCP server\n", len(mcps))
		for _, m := range mcps {
			fmt.Printf("    - %s (%s): %s %v\n", m.Name, m.Transport, m.Command, m.Args)
		}
	}
	fmt.Println()

	// 4. Skills
	fmt.Println("--- 4. ListSkills ---")
	if skills, err := adp.ListSkills(); err != nil {
		fmt.Printf("  ❌ 错误: %v\n", err)
	} else {
		fmt.Printf("  ✓ %d 个 skill\n", len(skills))
		for _, s := range skills {
			fmt.Printf("    - /%s: %s\n", s.Name, truncate(s.Description, 60))
		}
	}
	fmt.Println()

	// 5. Plugins
	fmt.Println("--- 5. ListPlugins ---")
	if plugs, err := adp.ListPlugins(); err != nil {
		fmt.Printf("  ❌ 错误: %v\n", err)
	} else {
		fmt.Printf("  ✓ %d 个 plugin (enabled)\n", len(plugs))
		for _, p := range plugs {
			fmt.Printf("    - %s\n", p.Name)
		}
	}
	fmt.Println()

	// 6. Plans
	fmt.Println("--- 6. ListPlans ---")
	if plans, err := adp.ListPlans(); err != nil {
		fmt.Printf("  ❌ 错误: %v\n", err)
	} else {
		fmt.Printf("  ✓ %d 个 plan file\n", len(plans))
		for _, p := range plans[:min(3, len(plans))] {
			fmt.Printf("    - %s (%d bytes, %s)\n", p.Name, p.Size, time.Unix(p.ModifiedAt, 0).Format("01-02 15:04"))
		}
	}
	fmt.Println()

	// 7. FileHistory
	fmt.Println("--- 7. ListFileHistory ---")
	if edits, err := adp.ListFileHistory(); err != nil {
		fmt.Printf("  ❌ 错误: %v\n", err)
	} else {
		fmt.Printf("  ✓ %d 条 file edit 历史 (最近 200)\n", len(edits))
		for _, e := range edits[:min(3, len(edits))] {
			fmt.Printf("    - %s (v%d, %s)\n", truncate(e.Path, 50), e.Version, time.Unix(e.Timestamp, 0).Format("01-02 15:04"))
		}
	}
	fmt.Println()

	// 8. DailyActivity
	fmt.Println("--- 8. DailyActivity ---")
	if daily, err := adp.DailyActivity(); err != nil {
		fmt.Printf("  ❌ 错误: %v\n", err)
	} else {
		fmt.Printf("  ✓ %d 天数据 (最近)\n", len(daily))
		total := 0
		for _, d := range daily {
			total += d.MessageCount
		}
		fmt.Printf("    总消息数: %d\n", total)
		for _, d := range daily[:min(5, len(daily))] {
			fmt.Printf("    %s: %d msg / %d sess / %d tools\n", d.Date, d.MessageCount, d.SessionCount, d.ToolCallCount)
		}
	}
	fmt.Println()

	// 9. ModelTokenUsage + 真实 USD
	fmt.Println("--- 9. ModelTokenUsage + 真实 USD ---")
	if usages, err := adp.ModelTokenUsage(); err != nil {
		fmt.Printf("  ❌ 错误: %v\n", err)
	} else {
		var totalIn, totalOut, totalCR, totalCW int64
		var totalCost float64
		modelSet := map[string]bool{}
		for _, u := range usages {
			totalIn += u.InputTokens
			totalOut += u.OutputTokens
			totalCR += u.CacheRead
			totalCW += u.CacheWrite
			cost, known := tribes.CostFromUsage(u.Model, u.InputTokens, u.OutputTokens, u.CacheRead, u.CacheWrite)
			if known {
				totalCost += cost
			}
			modelSet[u.Model] = true
		}
		fmt.Printf("  ✓ %d 条 (model, date) 记录\n", len(usages))
		fmt.Printf("    涉及 model: %d 种\n", len(modelSet))
		for m := range modelSet {
			fmt.Printf("      - %s\n", m)
		}
		fmt.Printf("    总输入:  %.1f k tokens\n", float64(totalIn)/1000)
		fmt.Printf("    总输出:  %.1f k tokens\n", float64(totalOut)/1000)
		fmt.Printf("    缓存读:  %.1f k tokens\n", float64(totalCR)/1000)
		fmt.Printf("    缓存写:  %.1f k tokens\n", float64(totalCW)/1000)
		if totalCost > 0 {
			fmt.Printf("    已知价格 USD: $%.2f\n", totalCost)
		} else {
			fmt.Printf("    已知价格 USD: n/a（自定义 model 不在价格表）\n")
		}
		for _, u := range usages[:min(3, len(usages))] {
			cost, known := tribes.CostFromUsage(u.Model, u.InputTokens, u.OutputTokens, u.CacheRead, u.CacheWrite)
			costStr := "n/a"
			if known {
				costStr = fmt.Sprintf("$%.4f", cost)
			}
			fmt.Printf("    %s %s: in=%.0f out=%.0f cost=%s\n", u.Date, truncate(u.Model, 20), float64(u.InputTokens), float64(u.OutputTokens), costStr)
		}
	}
	fmt.Println()

	// 10. ParseSessions
	fmt.Println("--- 10. ParseSessions ---")
	if shards, err := adp.ParseSessions(); err != nil {
		fmt.Printf("  ❌ 错误: %v\n", err)
	} else {
		fmt.Printf("  ✓ %d 个 session shard\n", len(shards))
		var totalTokens int64
		for _, s := range shards {
			totalTokens += s.TokenCount
		}
		fmt.Printf("    总 token: %.1f k\n", float64(totalTokens)/1000)
		for _, s := range shards[:min(3, len(shards))] {
			fmt.Printf("    - %s: %s\n", s.ID[:8], truncate(s.Summary, 60))
			fmt.Printf("      model=%s msgs=%d tokens=%d project=%s\n", s.Model, s.MessageCount, s.TokenCount, truncate(s.Project, 50))
		}
	}
	fmt.Println()

	// 11. RecentSlashCommands
	fmt.Println("--- 11. RecentSlashCommands ---")
	if slash, err := adp.RecentSlashCommands(10); err != nil {
		fmt.Printf("  ❌ 错误: %v\n", err)
	} else {
		fmt.Printf("  ✓ 最近 %d 条 slash command\n", len(slash))
		for _, c := range slash {
			fmt.Printf("    %s %s\n", c.Command, truncate(c.Args, 40))
		}
	}
	fmt.Println()

	// 12. ReadSession (取最新 session 完整)
	fmt.Println("--- 12. ReadSession (最新 session 全文) ---")
	if shards, _ := adp.ParseSessions(); len(shards) > 0 {
		// 找最新
		latest := shards[0]
		for _, s := range shards {
			if s.Timestamp > latest.Timestamp {
				latest = s
			}
		}
		fmt.Printf("  读 session: %s\n", latest.ID)
		if evs, err := adp.ReadSession(latest.ID); err != nil {
			fmt.Printf("  ❌ 错误: %v\n", err)
		} else {
			fmt.Printf("  ✓ %d 个 event\n", len(evs))
			// 统计
			types := map[string]int{}
			for _, e := range evs {
				types[e.Type]++
			}
			for t, c := range types {
				fmt.Printf("    %s: %d\n", t, c)
			}
			// 展示最近 3 个
			for _, e := range evs[max(0, len(evs)-3):] {
				role := e.Role
				if role == "" {
					role = e.Type
				}
				body := e.Content
				if body == "" {
					body = e.Thinking
				}
				if body == "" && e.Tool != nil {
					body = fmt.Sprintf("[tool: %s]", e.Tool.Name)
				}
				fmt.Printf("    %s: %s\n", role, truncate(body, 80))
			}
		}
	}
	fmt.Println()

	// 13. StreamLatest（启动后看 5s 内有没有 event）
	fmt.Println("--- 13. StreamLatest (监听 5 秒) ---")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	count := 0
	err := adp.StreamLatest(ctx, func(ev tribes.SessionEvent) {
		count++
		if count <= 5 {
			body := ev.Content
			if body == "" {
				body = ev.Thinking
			}
			if body == "" && ev.Tool != nil {
				body = fmt.Sprintf("[tool: %s]", ev.Tool.Name)
			}
			if body == "" && ev.Tokens != nil {
				body = fmt.Sprintf("[tokens: +%d/+%d]", ev.Tokens.Input, ev.Tokens.Output)
			}
			fmt.Printf("  [event] type=%s role=%s body=%s\n", ev.Type, ev.Role, truncate(body, 60))
		}
	})
	if err != nil {
		fmt.Printf("  ❌ StreamLatest 错误: %v\n", err)
	} else {
		fmt.Printf("  ✓ 5 秒内收到 %d 个 event（Claude Code 还在写 session 的话就会有）\n", count)
	}
	fmt.Println()

	fmt.Println("=== 验证完成 ===")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n < 3 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

func mask(v any) string {
	s := fmt.Sprintf("%v", v)
	if len(s) > 20 {
		return s[:10] + "••••" + s[len(s)-4:]
	}
	return strings.Repeat("•", len(s))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
