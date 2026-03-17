{{ if .Versions -}}
{{ if .Unreleased.CommitGroups -}}
## [Unreleased]
{{ range .Unreleased.CommitGroups -}}
### {{ .Title }}
{{ range .Commits -}}
* {{ if .Scope }}**{{ .Scope }}**: {{ end }}{{ .Subject }}
{{ end -}}
{{ end -}}
{{- if .Unreleased.NoteGroups -}}
{{ range .Unreleased.NoteGroups -}}
### {{ .Title }}
{{ range .Notes }}
{{ .Body }}
{{ end }}
{{ end -}}
{{ end -}}
{{ end -}}
{{ end -}}

{{ range .Versions }}
## {{ .Tag.Name }} ({{ datetime "2006-01-02" .Tag.Date }})
{{ range .CommitGroups -}}
### {{ .Title }}
{{ range .Commits -}}
* {{ if .Scope }}**{{ .Scope }}**: {{ end }}{{ .Subject }}
{{ end -}}
{{ end -}}
{{- if .NoteGroups -}}
{{ range .NoteGroups -}}
### {{ .Title }}
{{ range .Notes }}
{{ .Body }}
{{ end -}}
{{ end -}}
{{ end -}}
{{ end -}}
