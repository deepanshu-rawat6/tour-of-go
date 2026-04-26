package report

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/tour-of-go/aws-resource-reaper/internal/rules"
)

// Reporter formats findings as text table or JSON.
type Reporter struct {
	Format string // "text" or "json"
}

// Print writes findings to w in the configured format.
func (r *Reporter) Print(findings []rules.Finding, w io.Writer) error {
	if r.Format == "json" {
		return printJSON(findings, w)
	}
	return printText(findings, w)
}

func printText(findings []rules.Finding, w io.Writer) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ACCOUNT\tREGION\tTYPE\tID\tACTION\tESTIMATED SAVINGS\tREASON")
	fmt.Fprintln(tw, "-------\t------\t----\t--\t------\t-----------------\t------")

	var totalSavings float64
	for _, f := range findings {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t$%.2f/mo\t%s\n",
			f.Resource.AccountID,
			f.Resource.Region,
			f.Resource.Type,
			f.Resource.ID,
			f.Action,
			f.EstimatedMonthlySavings,
			f.Reason,
		)
		totalSavings += f.EstimatedMonthlySavings
	}
	tw.Flush()

	fmt.Fprintf(w, "\nTotal findings: %d | Estimated monthly savings: $%.2f\n", len(findings), totalSavings)
	return nil
}

func printJSON(findings []rules.Finding, w io.Writer) error {
	type summary struct {
		Findings              []rules.Finding `json:"findings"`
		TotalFindings         int             `json:"total_findings"`
		EstimatedMonthlySavings float64       `json:"estimated_monthly_savings"`
	}
	var total float64
	for _, f := range findings {
		total += f.EstimatedMonthlySavings
	}
	return json.NewEncoder(w).Encode(summary{
		Findings:              findings,
		TotalFindings:         len(findings),
		EstimatedMonthlySavings: total,
	})
}
