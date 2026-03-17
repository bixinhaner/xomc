function scanClick() {
    $('#uploadFile').click();
}

function submitForm() {
    var files = document.getElementById("uploadFile").files;
    var filePath = $("#uploadFile").val();
    $("#uploadForm").form('submit', {});
}

function successFormatter(value, rowData, rowIndex) {
    if (value == "true") {// 连接断开
        return "Yes";
    } else {
        return "No";
    }
}

function successStyler(value, rowData, rowIndex) {
    if (value == "false") {
        return "background:rgba(187, 52, 54, 0.99)";
    }
}

$(function () {
    $("#uploadFile").bind("change", function () {
        $("#txt_paramFilePath").val(this.value);
        submitForm();
    });
});