var smsWS;
var regWS;
var incomingTable;
var sendForm;
var registeredSelect;
var registerForm;

$.fn.serializeObject = function() {
    var o = {};
    var a = this.serializeArray();
    $.each(a, function() {
        if (o[this.name] !== undefined) {
            if (!o[this.name].push) {
                o[this.name] = [o[this.name]];
            }
            o[this.name].push(this.value || '');
        } else {
            o[this.name] = this.value || '';
        }
    });
    return o;
};
$(document).ready(function() {
    incomingTable = $("table#incoming_sms tbody");
    sendForm = $("form#send_sms");
    registeredSelect = $("select#registered");
    registerForm = $("form#register");

    $("select#registered").change(function() {
        $("input#Number").val($(this).val());
    });

    $("button#regbutton").click(function() {
        register();
    });

    $("button#mrxbutton").click(function() {
        o = $("form#register").serializeObject();
        $.post("setmrx", JSON.stringify(o));
    });

    smsWS = new WebSocket("ws://localhost:8080/sms");
    smsWS.onmessage = function (e) {
        var sms = JSON.parse(e.data);
        row = $('<tr><td class="time">' + sms.Time + '</td><td class="number">' + sms.Number + '</td><td class="body">' + sms.Body + "</td></tr>");
        incomingTable.prepend(row);
    };
    smsWS.onclose = function () {
        console.log("smsWS closed");
    };

    regWS = new WebSocket("ws://localhost:8080/reg");
    regWS.onmessage = function (e) {
        console.log(e);
        var registered = JSON.parse(e.data);
        for (i in registered) {
            option = $('<option class="register">' + registered[i] + '</option>');
            registeredSelect.append(option);
        }
    }
    regWS.onclose = function () {
        console.log("regWS closed");
    }
});

function sendSMS() {
    o = sendForm.serializeObject();
    smsWS.send(JSON.stringify(o));
}

function register() {
    o = registerForm.serializeObject();
    regWS.send(JSON.stringify(o));
}
