package templates

var PageTpl = `
{{- define "page" }}
<!DOCTYPE html>
<html>
    {{- template "header" . }}
<body>

<style>
.box { justify-content:center; display:flex; flex-wrap:wrap }
.footer {
  position: fixed;
  bottom: 0px;
  padding-top: 5px;
  border-top: 1px solid gray;
  width: 100%;
  background-color: #f5f5f5;
  font-size: 14px;
  text-align: center;
}

</style>
<div class="box"> {{- range .Charts }} {{ template "base" . }} {{- end }} </div>

<div class="footer">"
  <div class="container">
	© wows-stat. All rights reserved | © Wargaming.net. All rights reserved | This is application is a third-party tool under Wargaming Developers Program.
  </div>
</div>
</body>
</html>
{{ end }}
`
