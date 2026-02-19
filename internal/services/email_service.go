package services

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/gomail.v2"
)

// SendMissedSIPAlert sends email alert for missed SIP installments
func SendMissedSIPAlert(missedSIPs []MissedSIP) error {
	if len(missedSIPs) == 0 {
		return nil
	}

	smtpHost := os.Getenv("SMTP_HOST")
	smtpPortStr := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	alertEmail := os.Getenv("ALERT_EMAIL")

	if smtpHost == "" || smtpPortStr == "" || smtpUser == "" || smtpPassword == "" || alertEmail == "" {
		return fmt.Errorf("email configuration not set in environment variables")
	}

	smtpPort, err := strconv.Atoi(smtpPortStr)
	if err != nil {
		return fmt.Errorf("invalid SMTP port: %w", err)
	}

	// Build email body
	var body strings.Builder
	body.WriteString("<html><body>")
	body.WriteString("<h2>Missed SIP Installments Alert</h2>")
	body.WriteString(fmt.Sprintf("<p>The following %d SIP installments have been missed:</p>", len(missedSIPs)))
	body.WriteString("<table border='1' cellpadding='8' cellspacing='0' style='border-collapse: collapse;'>")
	body.WriteString("<tr style='background-color: #f0f0f0;'>")
	body.WriteString("<th>Client Name</th><th>Fund Name</th><th>Expected Date</th><th>Amount</th><th>Days Missed</th>")
	body.WriteString("</tr>")

	for _, sip := range missedSIPs {
		body.WriteString("<tr>")
		body.WriteString(fmt.Sprintf("<td>%s</td>", sip.ClientName))
		body.WriteString(fmt.Sprintf("<td>%s</td>", sip.FundName))
		body.WriteString(fmt.Sprintf("<td>%s</td>", sip.ExpectedDate.Format("02-Jan-2006")))
		body.WriteString(fmt.Sprintf("<td>₹%.2f</td>", sip.Amount))
		body.WriteString(fmt.Sprintf("<td style='color: red;'><b>%d days</b></td>", sip.DaysMissed))
		body.WriteString("</tr>")
	}

	body.WriteString("</table>")
	body.WriteString("<p><i>This is an automated alert from your Mutual Fund Dashboard.</i></p>")
	body.WriteString("</body></html>")

	// Create email message
	m := gomail.NewMessage()
	m.SetHeader("From", smtpUser)
	m.SetHeader("To", alertEmail)
	m.SetHeader("Subject", fmt.Sprintf("SIP Alert: %d Missed Installments", len(missedSIPs)))
	m.SetBody("text/html", body.String())

	// Send email
	d := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPassword)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// SendCustomEmail sends a custom email (can be used for other notifications)
func SendCustomEmail(to, subject, htmlBody string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPortStr := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPassword := os.Getenv("SMTP_PASSWORD")

	if smtpHost == "" || smtpPortStr == "" || smtpUser == "" || smtpPassword == "" {
		return fmt.Errorf("email configuration not set")
	}

	smtpPort, err := strconv.Atoi(smtpPortStr)
	if err != nil {
		return fmt.Errorf("invalid SMTP port: %w", err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", smtpUser)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)

	d := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPassword)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
