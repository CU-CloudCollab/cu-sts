// Package profile provides functions accessing profiles in cu-sts config file.
package profile

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/spf13/viper"
)

// A Profile represents a single profile from the config file.
type Profile struct {
	Name       string
	Account    string `mapstructure:"account"`
	Role       string `mapstructure:"role"`
	IDProvider string `mapstructure:"id_provider"`
	Duration   int    `mapstructure:"duration"`
}

// Profiles returns all profiles from the loaded viper config file.
func Profiles() map[string]interface{} {
	return viper.GetStringMap("profile")
}

// New returns an "empty" Profile with defailt IDProvider and Duration values.
func New() Profile {
	return Profile{
		IDProvider: "cornell_idp",
		Duration:   3600,
	}
}

// NewFromConfig returns a Profile with values set from default, config, or flags.
func NewFromConfig(name string) (Profile, error) {
	p := New()
	p.Name = name

	if _, ok := Profiles()[name]; !ok {
		return p, fmt.Errorf("unable to find profile %s in config", name)
	}
	sectionKey := fmt.Sprintf("profile.%s", name)
	section := viper.Sub(sectionKey)
	err := section.Unmarshal(&p)
	if err != nil {
		return p, fmt.Errorf("unable to decode %s into struct: %v", name, err)
	}

	// Override values from config w/ flag since unmarshalling from a viper sub
	// doesn't get the dot-path overrides from flag binding. So we have to check
	// if the struct has changed AND if the flag-set value is non-default.
	// https://github.com/spf13/viper/issues/307
	if viper.GetInt("duration") != 3600 {
		p.Duration = viper.GetInt("duration")
	}

	if viper.GetString("id_provider") != "cornell_idp" {
		p.IDProvider = viper.GetString("id_provider")
	}

	if err = p.Validate(); err != nil {
		return p, fmt.Errorf("error validating profile %s: %v", name, err)
	}

	return p, nil
}

// Validate ensures a Profile's Account and Role are set.
func (p *Profile) Validate() error {
	if p.Account == "" {
		return fmt.Errorf(`missing required key "account"`)
	}
	if p.Role == "" {
		return fmt.Errorf(`missing required key "role"`)
	}
	return nil
}

// Credentials requires a base-64 SAMLAssertion and returns AWS sts.Credentials
// using the Profile's role, idprovider, etc.
func (p *Profile) Credentials(samlAssertion string) (*sts.Credentials, error) {
	principalArn := fmt.Sprintf("arn:aws:iam::%s:saml-provider/%s", p.Account, p.IDProvider)
	roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", p.Account, p.Role)
	durationI64 := int64(p.Duration)

	svc := sts.New(session.New())
	input := &sts.AssumeRoleWithSAMLInput{
		PrincipalArn:    aws.String(principalArn),
		RoleArn:         aws.String(roleArn),
		DurationSeconds: &durationI64,
		SAMLAssertion:   &samlAssertion,
	}
	resp, err := svc.AssumeRoleWithSAML(input)
	if err != nil {
		return nil, err
	}
	return resp.Credentials, nil
}
