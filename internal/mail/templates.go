package mail

import "fmt"

// NewMessageNotificationBody returns an HTML email body notifying the domain owner
// that a new message has been submitted through their contact form.
func NewMessageNotificationBody(domainName, senderName, senderEmail, messageBody string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background-color: #f4f4f7; margin: 0; padding: 0; }
    .container { max-width: 600px; margin: 40px auto; background-color: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.08); }
    .header { background-color: #1a1a2e; color: #ffffff; padding: 24px 32px; }
    .header h1 { margin: 0; font-size: 20px; font-weight: 600; }
    .body { padding: 32px; color: #333333; line-height: 1.6; }
    .meta { margin-bottom: 24px; }
    .meta p { margin: 4px 0; font-size: 14px; color: #555555; }
    .meta strong { color: #333333; }
    .message-box { background-color: #f8f9fa; border-left: 4px solid #1a1a2e; padding: 16px 20px; border-radius: 0 4px 4px 0; white-space: pre-wrap; word-wrap: break-word; font-size: 14px; color: #333333; }
    .footer { padding: 20px 32px; text-align: center; font-size: 12px; color: #999999; border-top: 1px solid #eeeeee; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>New Message on %s</h1>
    </div>
    <div class="body">
      <div class="meta">
        <p><strong>From:</strong> %s</p>
        <p><strong>Email:</strong> %s</p>
      </div>
      <div class="message-box">%s</div>
    </div>
    <div class="footer">
      This notification was sent by DeadDrop on behalf of %s.
    </div>
  </div>
</body>
</html>`, domainName, senderName, senderEmail, messageBody, domainName)
}
