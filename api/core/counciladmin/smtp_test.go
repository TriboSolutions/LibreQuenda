package counciladmin

import "testing"

func TestSmtpInputValidate(t *testing.T) {
	var valid = SmtpInput{Host: " smtp.lalin.gal ", Username: "reservas", From: "Concello de Lalín <reservas@lalin.gal>"}
	var with = func(change func(*SmtpInput)) SmtpInput {
		var in = valid
		change(&in)
		return in
	}
	var cases = []struct {
		name  string
		input SmtpInput
		want  string
	}{
		{"valid, port defaults to 587", valid, ""},
		{"no host", with(func(in *SmtpInput) { in.Host = " " }), "errors.invalid_smtp_host"},
		{"host with port", with(func(in *SmtpInput) { in.Host = "smtp.lalin.gal:587" }), "errors.invalid_smtp_host"},
		{"host as URL", with(func(in *SmtpInput) { in.Host = "smtp://smtp.lalin.gal" }), "errors.invalid_smtp_host"},
		{"port out of range", with(func(in *SmtpInput) { in.Port = 70000 }), "errors.invalid_smtp_port"},
		{"bare sender", with(func(in *SmtpInput) { in.From = "reservas@lalin.gal" }), ""},
		{"bad sender", with(func(in *SmtpInput) { in.From = "reservas" }), "errors.invalid_smtp_from"},
	}
	for _, c := range cases {
		var input = c.input
		if got := input.validate(); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
	var input = valid
	input.validate()
	if input.Port != 587 || input.Host != "smtp.lalin.gal" {
		t.Errorf("normalised: got port %d, host %q", input.Port, input.Host)
	}
}
