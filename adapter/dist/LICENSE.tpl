========================================================================
Metric Adapter related licenses
========================================================================

The following components are used in the metric adapter.See project link for details.
The text of each license is also included at licenses/adapter-licenses/LICENSE-[project].txt.
{{ range .Groups }}
========================================================================
{{.LicenseID}} licenses
========================================================================
{{range .Deps}}
    {{.Name}} {{.Version}} {{.LicenseID}}
{{- end }}
{{ end }}
