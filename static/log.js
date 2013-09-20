var logWS;
var logTable;
var logTypes = [ "", "warning", "error", "fatal" ];
$(document).ready(function() {
    logTable = $("table#log tbody")
    logWS = new WebSocket("ws://localhost:8080/log");
    logWS.onmessage = function (e) {
        var line = JSON.parse(e.data);
        row = $("<tr class=\"" + logTypes[line.Type] + "\"><td>" + line.Time + "</td><td>" + line.Text + "</td></tr>");
        logTable.prepend(row);
    };
    logWS.onclose = function () {
        console.log("Log Closed");
    };
});

