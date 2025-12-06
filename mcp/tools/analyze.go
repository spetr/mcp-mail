package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spetr/mcp-mail/memory"
	"github.com/spetr/mcp-mail/types"
)

func (r *Registry) registerAnalyzeTools() {
	// account_analyze - Deep analysis of account
	r.addTool(
		mcp.NewTool("account_analyze",
			mcp.WithDescription("Perform deep analysis of an email account. Analyzes messages to detect patterns, identify important senders, and build an account profile. Uses analysis_depth and analysis_period from preferences (default: 1000 messages, 180 days)."),
			requiredString("account_id", "Account ID"),
			optionalNumber("depth", "Override: max messages to analyze"),
			optionalNumber("period", "Override: days to look back"),
			optionalBool("include_sent", "Also analyze Sent folder (default true)"),
		),
		r.handleAccountAnalyze,
	)
}

// AnalysisResult contains the results of account analysis
type AnalysisResult struct {
	Analyzed struct {
		Messages   int      `json:"messages"`
		PeriodDays int      `json:"period_days"`
		Folders    []string `json:"folders"`
	} `json:"analyzed"`
	Findings struct {
		TotalSenders          int             `json:"total_senders"`
		TopSenders            []SenderSummary `json:"top_senders"`
		HighResponseRate      []SenderSummary `json:"high_response_rate"`
		DetectedNewsletters   []string        `json:"detected_newsletters"`
		DetectedNotifications []string        `json:"detected_notifications"`
	} `json:"findings"`
	Suggestions    []string `json:"suggestions"`
	ProfileUpdated bool     `json:"profile_updated"`
}

// SenderSummary contains summary info about a sender
type SenderSummary struct {
	Email        string  `json:"email"`
	Name         string  `json:"name,omitempty"`
	Count        int     `json:"count"`
	ResponseRate float64 `json:"response_rate,omitempty"`
	Type         string  `json:"type,omitempty"`
}

func (r *Registry) handleAccountAnalyze(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	accountID := request.GetString("account_id", "")
	depthOverride := request.GetInt("depth", 0)
	periodOverride := request.GetInt("period", 0)
	includeSent := request.GetBool("include_sent", true)

	// Get client
	client, err := r.imapMgr.GetClient(accountID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get client: %v", err)), nil
	}

	// Get analysis preferences
	depth, period, _ := r.memoryMgr.GetAnalysisPreferences(accountID)
	if depthOverride > 0 {
		depth = depthOverride
	}
	if periodOverride > 0 {
		period = periodOverride
	}

	// Calculate date cutoff
	cutoffDate := time.Now().AddDate(0, 0, -period)

	// Folders to analyze
	folders := []string{"INBOX"}
	if includeSent {
		// Try common sent folder names
		sentFolders := []string{"Sent", "Sent Items", "Sent Messages", "[Gmail]/Sent Mail"}
		folderList, err := client.ListFolders()
		if err == nil {
			for _, sf := range sentFolders {
				for _, f := range folderList {
					if strings.EqualFold(f.Name, sf) || strings.EqualFold(f.Name, "[Gmail]/Sent Mail") {
						folders = append(folders, f.Name)
						break
					}
				}
			}
		}
	}

	// Collect sender statistics
	senderStats := make(map[string]*memory.SenderProfile)
	totalMessages := 0

	const batchSize = 100 // Fetch headers in batches of 100

	for _, folder := range folders {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return mcp.NewToolResultError("analysis cancelled"), nil
		default:
		}

		// Search for messages within period
		criteria := &types.SearchCriteria{
			Since: &cutoffDate,
		}

		searchResult, err := client.Search(folder, criteria)
		if err != nil {
			continue // Skip folders with errors
		}

		uids := searchResult.UIDs

		// Limit to depth
		if len(uids) > depth {
			uids = uids[len(uids)-depth:]
		}

		// Calculate how many messages we can process from this folder
		maxToProcess := depth - totalMessages
		if maxToProcess <= 0 {
			break
		}
		if len(uids) > maxToProcess {
			uids = uids[len(uids)-maxToProcess:]
		}

		// Process UIDs in batches for efficiency
		for i := 0; i < len(uids); i += batchSize {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				return mcp.NewToolResultError("analysis cancelled"), nil
			default:
			}

			end := i + batchSize
			if end > len(uids) {
				end = len(uids)
			}
			batchUIDs := uids[i:end]

			// Batch fetch headers
			messages, err := client.GetMessageHeadersBatch(folder, batchUIDs)
			if err != nil {
				continue // Skip failed batches
			}

			for _, msg := range messages {
				if msg == nil {
					continue
				}

				totalMessages++

				// Extract sender from first From address
				if len(msg.From) == 0 {
					continue
				}
				fromAddr := msg.From[0]
				email := strings.ToLower(fromAddr.Address)
				if email == "" {
					continue
				}

				sp, exists := senderStats[email]
				if !exists {
					sp = &memory.SenderProfile{
						Email: email,
						Name:  fromAddr.Name,
						Stats: memory.SenderStats{
							FirstSeen: msg.Date,
						},
					}
					senderStats[email] = sp
				}

				sp.Stats.TotalSeen++
				if msg.Date.After(sp.Stats.LastSeen) {
					sp.Stats.LastSeen = msg.Date
				}
				if msg.Date.Before(sp.Stats.FirstSeen) {
					sp.Stats.FirstSeen = msg.Date
				}

				// Check flags
				for _, flag := range msg.Flags {
					switch flag {
					case "\\Answered":
						sp.Stats.TotalAnswered++
					case "\\Flagged":
						sp.Stats.TotalFlagged++
					}
				}

				// Detect type from address patterns
				if sp.Type == "" {
					sp.Type = detectSenderTypeFromAddress(email, msg.Subject)
				}
			}
		}

		// Limit total messages
		if totalMessages >= depth {
			break
		}
	}

	// Build results
	result := AnalysisResult{}
	result.Analyzed.Messages = totalMessages
	result.Analyzed.PeriodDays = period
	result.Analyzed.Folders = folders
	result.Findings.TotalSenders = len(senderStats)

	// Sort senders by count
	type senderCount struct {
		email string
		sp    *memory.SenderProfile
	}
	var sortedSenders []senderCount
	for email, sp := range senderStats {
		sortedSenders = append(sortedSenders, senderCount{email, sp})
	}
	sort.Slice(sortedSenders, func(i, j int) bool {
		return sortedSenders[i].sp.Stats.TotalSeen > sortedSenders[j].sp.Stats.TotalSeen
	})

	// Top 10 senders
	for i := 0; i < len(sortedSenders) && i < 10; i++ {
		sc := sortedSenders[i]
		var responseRate float64
		if sc.sp.Stats.TotalSeen > 0 {
			responseRate = float64(sc.sp.Stats.TotalAnswered) / float64(sc.sp.Stats.TotalSeen) * 100
		}
		result.Findings.TopSenders = append(result.Findings.TopSenders, SenderSummary{
			Email:        sc.email,
			Name:         sc.sp.Name,
			Count:        sc.sp.Stats.TotalSeen,
			ResponseRate: responseRate,
			Type:         sc.sp.Type,
		})
	}

	// High response rate senders (>50% and at least 3 messages)
	for _, sc := range sortedSenders {
		if sc.sp.Stats.TotalSeen >= 3 && sc.sp.Stats.TotalAnswered > 0 {
			rate := float64(sc.sp.Stats.TotalAnswered) / float64(sc.sp.Stats.TotalSeen)
			if rate >= 0.5 {
				result.Findings.HighResponseRate = append(result.Findings.HighResponseRate, SenderSummary{
					Email:        sc.email,
					Name:         sc.sp.Name,
					Count:        sc.sp.Stats.TotalSeen,
					ResponseRate: rate * 100,
				})
			}
		}
	}
	// Limit to top 10
	if len(result.Findings.HighResponseRate) > 10 {
		result.Findings.HighResponseRate = result.Findings.HighResponseRate[:10]
	}

	// Detected newsletters and notifications
	for _, sc := range sortedSenders {
		if sc.sp.Type == "newsletter" {
			result.Findings.DetectedNewsletters = append(result.Findings.DetectedNewsletters, sc.email)
		} else if sc.sp.Type == "notification" {
			result.Findings.DetectedNotifications = append(result.Findings.DetectedNotifications, sc.email)
		}
	}

	// Generate suggestions
	topCount := 5
	if len(sortedSenders) < topCount {
		topCount = len(sortedSenders)
	}
	for _, sc := range sortedSenders[:topCount] {
		if sc.sp.Stats.TotalSeen >= 5 {
			rate := float64(sc.sp.Stats.TotalAnswered) / float64(sc.sp.Stats.TotalSeen)
			if rate >= 0.7 {
				result.Suggestions = append(result.Suggestions,
					fmt.Sprintf("%s má %.0f%% response rate - doporučuji označit jako 'high' importance", sc.email, rate*100))
			}
		}
	}

	if len(result.Findings.DetectedNewsletters) > 5 {
		result.Suggestions = append(result.Suggestions,
			fmt.Sprintf("Nalezeno %d newsletterů - zvaž označení jako 'ignore' pomocí memory_set_sender", len(result.Findings.DetectedNewsletters)))
	}

	// Update memory with collected stats - only store significant senders
	err = r.memoryMgr.Update(accountID, func(mem *memory.AccountMemory) {
		if mem.Profile.SenderProfiles == nil {
			mem.Profile.SenderProfiles = make(map[string]*memory.SenderProfile)
		}

		// Only store senders that are significant:
		// - Already classified (has importance, relationship, or notes)
		// - Have >= 3 emails
		// - Have response rate >= 30%
		// - Are flagged
		for email, sp := range senderStats {
			existing, exists := mem.Profile.SenderProfiles[email]

			// Always update existing profiles (preserve user classifications)
			if exists {
				existing.Stats = sp.Stats
				if existing.Name == "" && sp.Name != "" {
					existing.Name = sp.Name
				}
				if existing.Type == "" && sp.Type != "" {
					existing.Type = sp.Type
				}
				continue
			}

			// For new senders, only store if significant
			isSignificant := false

			// Has enough emails
			if sp.Stats.TotalSeen >= 3 {
				isSignificant = true
			}

			// Has response activity (user replied)
			if sp.Stats.TotalAnswered > 0 {
				isSignificant = true
			}

			// Is flagged
			if sp.Stats.TotalFlagged > 0 {
				isSignificant = true
			}

			// Has detected type (newsletter, notification, service)
			if sp.Type != "" {
				isSignificant = true
			}

			if isSignificant {
				mem.Profile.SenderProfiles[email] = sp
			}
		}

		// Mark as analyzed
		mem.Profile.LastAnalyzed = time.Now()
		mem.Profile.AnalyzedCount = totalMessages
	})

	result.ProfileUpdated = err == nil

	// Return results
	resultJSON, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// detectSenderTypeFromAddress detects sender type from email address and subject
func detectSenderTypeFromAddress(email, subject string) string {
	emailLower := strings.ToLower(email)
	subjectLower := strings.ToLower(subject)

	// Newsletter patterns
	if strings.Contains(emailLower, "newsletter") ||
		strings.Contains(emailLower, "noreply") ||
		strings.Contains(emailLower, "no-reply") ||
		strings.Contains(emailLower, "digest") ||
		strings.Contains(emailLower, "news@") ||
		strings.Contains(emailLower, "updates@") ||
		strings.Contains(subjectLower, "unsubscribe") ||
		strings.Contains(subjectLower, "newsletter") {
		return "newsletter"
	}

	// Notification patterns
	if strings.Contains(emailLower, "notification") ||
		strings.Contains(emailLower, "alert") ||
		strings.Contains(emailLower, "notify") ||
		strings.Contains(emailLower, "mailer-daemon") ||
		strings.Contains(emailLower, "postmaster") {
		return "notification"
	}

	// Service patterns
	if strings.Contains(emailLower, "support@") ||
		strings.Contains(emailLower, "help@") ||
		strings.Contains(emailLower, "info@") ||
		strings.Contains(emailLower, "service@") {
		return "service"
	}

	return ""
}
