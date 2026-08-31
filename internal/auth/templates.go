package auth

import (
	"bytes"
	"fmt"
	"html/template"
)

// EmailTemplateType defines supported email notification templates.
type EmailTemplateType string

const (
	TemplateWelcome       EmailTemplateType = "welcome"
	TemplateOTP           EmailTemplateType = "otp"
	TemplatePasswordReset EmailTemplateType = "password_reset"
	TemplateSecurityAlert EmailTemplateType = "security_alert"
)

type TemplateData struct {
	PlatformName string
	UserName     string
	ActionURL    string
	OTPCode      string
	IPAddress    string
	UserAgent    string
	SupportEmail string
	Year         int
}

const htmlBaseTemplate = `
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #0F172A; color: #F8FAFC; margin: 0; padding: 24px; }
  .card { max-width: 520px; margin: 0 auto; background-color: #1E293B; border-radius: 12px; padding: 32px; border: 1px solid #334155; }
  .brand { font-size: 20px; font-weight: 800; color: #FF8A00; letter-spacing: -0.5px; margin-bottom: 24px; text-transform: uppercase; }
  h1 { font-size: 20px; font-weight: 700; color: #FFF; margin-top: 0; }
  p { font-size: 15px; line-height: 1.6; color: #94A3B8; }
  .otp-badge { display: inline-block; font-size: 28px; font-weight: 800; letter-spacing: 6px; color: #FF8A00; background: rgba(255, 138, 0, 0.1); padding: 12px 24px; border-radius: 8px; margin: 20px 0; border: 1px dashed #FF8A00; }
  .btn { display: inline-block; background: #FF8A00; color: #FFF; text-decoration: none; padding: 12px 24px; border-radius: 6px; font-weight: 700; font-size: 15px; margin: 20px 0; }
  .footer { margin-top: 32px; padding-top: 16px; border-top: 1px solid #334155; font-size: 12px; color: #64748B; text-align: center; }
</style>
</head>
<body>
  <div class="card">
    <div class="brand">{{.PlatformName}}</div>
    {{.Content}}
    <div class="footer">
      © {{.Year}} {{.PlatformName}}. सर्वाधिकार सुरक्षित। DPDP Act 2023 Compliant.
    </div>
  </div>
</body>
</html>
`

// RenderTemplate generates HTML email content for the specified template.
func RenderTemplate(templateType EmailTemplateType, data TemplateData) (string, error) {
	if data.PlatformName == "" {
		data.PlatformName = "BharatVani News Platform"
	}
	if data.SupportEmail == "" {
		data.SupportEmail = "support@newsplatform.in"
	}
	if data.Year == 0 {
		data.Year = 2026
	}

	var content string
	switch templateType {
	case TemplateOTP:
		content = fmt.Sprintf(`
			<h1>आपका सत्यापन कोड / Login OTP</h1>
			<p>नमस्ते %s,</p>
			<p>%s में लॉगिन करने के लिए आपका एकबारीय सत्यापन कोड (OTP) नीचे दिया गया है:</p>
			<div class="otp-badge">%s</div>
			<p>यह कोड 5 मिनट तक मान्य है। इसे किसी के साथ साझा न करें।</p>
		`, data.UserName, data.PlatformName, data.OTPCode)

	case TemplatePasswordReset:
		content = fmt.Sprintf(`
			<h1>पासवर्ड रीसेट अनुरोध / Password Reset</h1>
			<p>नमस्ते %s,</p>
			<p>आपके %s खाते का पासवर्ड रीसेट करने का अनुरोध प्राप्त हुआ है। नया पासवर्ड सेट करने के लिए नीचे दिए गए बटन पर क्लिक करें:</p>
			<a href="%s" class="btn">नया पासवर्ड बनाएं / Reset Password</a>
			<p>यदि आपने यह अनुरोध नहीं किया था, तो कृपया तुरंत अपने खाते की सुरक्षा जांचें।</p>
		`, data.UserName, data.PlatformName, data.ActionURL)

	case TemplateWelcome:
		content = fmt.Sprintf(`
			<h1>%s में आपका स्वागत है!</h1>
			<p>नमस्ते %s,</p>
			<p>भारत के सबसे उन्नत बहुभाषी व क्षेत्रीय समाचार मंच पर आपका पंजीकरण सफल रहा।</p>
			<p>अब आप 37 राज्य संस्करणों में निष्पक्ष और सटीक पत्रकारिता का अनुभव कर सकते हैं।</p>
		`, data.PlatformName, data.UserName)

	case TemplateSecurityAlert:
		content = fmt.Sprintf(`
			<h1>सुरक्षा चेतावनी / Security Alert</h1>
			<p>नमस्ते %s,</p>
			<p>आपके खाते में नए उपकरण या स्थान से लॉगिन किया गया है:</p>
			<p><b>IP Address:</b> %s<br><b>User Agent:</b> %s</p>
			<p>यदि यह गतिविधि आपने नहीं की है, तो तुरंत पासवर्ड बदलें।</p>
		`, data.UserName, data.IPAddress, data.UserAgent)
	}

	tmpl, err := template.New("base").Parse(htmlBaseTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, struct {
		PlatformName string
		Year         int
		Content      template.HTML
	}{
		PlatformName: data.PlatformName,
		Year:         data.Year,
		Content:      template.HTML(content),
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
