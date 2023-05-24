package templates

var PageTpl = `
{{- define "page" }}
<!DOCTYPE html>
<html>
    {{- template "header" . }}
<style>
	.box { 
	    justify-content:center;
            display:flex;
	    flex-wrap:wrap
        }

        /* Set up grid container */
        .container {
            display: grid;
            grid-template-rows: auto 1fr auto; /* Header, Content, Footer */
            min-height: 100vh;
        }

        /* Header styles */
        .header {
            background-color: #f5f5f5;;
            padding: 20px;
        }

        /* Content styles */
        .content {
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }

        /* Footer styles */
        .footer {
            background-color: #f5f5f5;;
            padding: 20px;
        }

</style>
<body>
    <div class="container">
        <header class="header">
	    <a href="https://eu.wows-stats.kakwalab.ovh" target="_blank">EU server stats</a> | <a href="https://na.wows-stats.kakwalab.ovh" target="_blank">NA server stats</a> | <a href="https://asia.wows-stats.kakwalab.ovh" target="_blank">ASIA server stats</a>
        </header>

        <div class="content">
	    <div class="box"> {{- range .Charts }} {{ template "base" . }} {{- end }} </div>
        </div>

        <footer class="footer">
		<p>© wows-stat. All rights reserved | © Wargaming.net. All rights reserved | This is application is a third-party tool under Wargaming Developers Program.</p>
        </footer>
    </div>
</body>
</html>
{{ end }}
`
