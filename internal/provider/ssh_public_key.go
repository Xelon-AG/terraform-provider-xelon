package provider

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"golang.org/x/crypto/ssh"
)

var (
	_ basetypes.StringTypable                    = (*sshPublicKeyType)(nil)
	_ basetypes.StringValuableWithSemanticEquals = (*sshPublicKey)(nil)
)

// sshPublicKeyType is an attribute type that represents SSH public key material
// with leading and trailing whitespace treated as semantically insignificant.
type sshPublicKeyType struct {
	basetypes.StringType
}

func (t sshPublicKeyType) String() string { return "xelon.sshPublicKeyType" }

func (t sshPublicKeyType) ValueType(_ context.Context) attr.Value { return &sshPublicKey{} }

func (t sshPublicKeyType) Equal(o attr.Type) bool {
	other, ok := o.(sshPublicKeyType)
	if !ok {
		return false
	}

	return t.StringType.Equal(other.StringType)
}

func (t sshPublicKeyType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return sshPublicKey{StringValue: in}, nil
}

func (t sshPublicKeyType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}

	stringValuable, diags := t.ValueFromString(ctx, stringValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting StringValue to StringValuable: %v", diags)
	}
	return stringValuable, nil
}

// sshPublicKey represents an SSH public key material.
type sshPublicKey struct {
	basetypes.StringValue
}

func (v sshPublicKey) Type(_ context.Context) attr.Type { return sshPublicKeyType{} }

func (v sshPublicKey) Equal(o attr.Value) bool {
	other, ok := o.(sshPublicKey)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

// StringSemanticEquals returns true if the given SSH public key string value is semantically equal
// to the current SSH public key string value. This comparison utilizes ssh.ParseAuthorizedKey and
// then compares the resulting key, comment and options representations.
func (v sshPublicKey) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(sshPublicKey)
	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			fmt.Sprintf("Expected SSH public key value type %T, got %T."+
				" Please report this issue to the provider developers.", v, newValuable),
		)
		return false, diags
	}

	newSSHPublicKey, newSSHPublicKeyOK := parseSingleAuthorizedSSHKey(newValue.ValueString())
	currentSSHPublicKey, currentSSHPublicKeyOK := parseSingleAuthorizedSSHKey(v.ValueString())

	if newSSHPublicKeyOK && currentSSHPublicKeyOK {
		return bytes.Equal(currentSSHPublicKey.key.Marshal(), newSSHPublicKey.key.Marshal()) &&
			currentSSHPublicKey.comment == newSSHPublicKey.comment &&
			slices.Equal(currentSSHPublicKey.options, newSSHPublicKey.options), diags
	}

	// compatibility path only as fallback
	return strings.TrimSpace(v.ValueString()) == strings.TrimSpace(newValue.ValueString()), diags
}

func (v sshPublicKey) NormalizedValue() string {
	return strings.TrimSpace(v.ValueString())
}

type parsedAuthorizedKey struct {
	key     ssh.PublicKey
	comment string
	options []string
}

func parseSingleAuthorizedSSHKey(value string) (parsedAuthorizedKey, bool) {
	input := bytes.TrimSpace([]byte(value))
	if len(input) == 0 || bytes.ContainsAny(input, "\r\n") {
		return parsedAuthorizedKey{}, false
	}

	key, comment, options, rest, err := ssh.ParseAuthorizedKey(input)
	if err != nil || len(bytes.TrimSpace(rest)) != 0 {
		return parsedAuthorizedKey{}, false
	}
	return parsedAuthorizedKey{
		key:     key,
		comment: comment,
		options: options,
	}, true
}

func newSSHPublicKeyValue(value string) sshPublicKey {
	return sshPublicKey{StringValue: basetypes.NewStringValue(value)}
}
