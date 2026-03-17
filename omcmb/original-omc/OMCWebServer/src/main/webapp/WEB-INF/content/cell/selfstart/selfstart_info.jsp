<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<style>
.switch {
    width:50px;
    height:20px;
    padding:2px;
    border-radius: 30px;
    -webkit-border-radius:30px;
    -moz-border-radius:30px;
    background-color: #838383;
    position: relative;
    float: right;
    display:inline-block;
    margin-right:10px;
}	
.switchDisabled{
	width:60px;
	height:30px;
	position:absolute;
	right:25px;
	cursor:not-allowed;
}
.btnn {
    width:20px;
    height:20px;
    -webkit-border-radius:30px;
    -moz-border-radius:30px;
    border-radius:30px;
    background-color: #fff;
    position: absolute;
}

.config_success {
	width: 22px;
	height: 22px;
	background: url('${ctx}/css/images/bi/config_success.png') no-repeat 0 0;
}

.config_failure {
	width: 22px;
	height: 22px;
	background: url('${ctx}/css/images/bi/config_failure.png') no-repeat 0 0;
}

.config_no {
	width: 22px;
	height: 22px;
	background: url('${ctx}/css/images/bi/config_no.png') no-repeat 0 0;
}
.IconTitleTop .titleButtonText {
	top:-21px;
}
</style>

<!-- 右上角导入和导出按钮 -->
<div class="omcTitleButtonGroup IconTitleTop">
	<div class="omcTitleButtonGroupItem">
		<span class="titleButtonText"><%=rb.getString("DaoRu")%></span>
		<span class="circleBg import_circle" onmouseenter="showTipText(this)" onmouseleave="showTipText(this)" onclick="importSelfstartFile()"></span>
	</div>
	<div class="omcTitleButtonGroupItem">
		<span class="titleButtonText"><%=rb.getString("DaoChu")%></span>
		<span class="circleBg export_circle" onmouseenter="showTipText(this)" onmouseleave="showTipText(this)" onclick="exportSelfstartFile()"></span>
	</div>
</div>

<!--  配置规划界面  -->
<div class="panelDefault" id="selfstartInfo">
	<div class="omcPageTitleDiv">
		<ul class="omcPageTitleContainer">
			<li class="default"><%=rb.getString("CanShuGuiHuaLieBiao")%></li>
		</ul>
	</div>
	<div class="panelTableDiv">
		<table class="easyui-datagrid" id="tableSelfStartParams" fit="true" data-options="border:false,fitColumns:true,singleSelect: true,
	                    rownumbers:true,url:'${ctx}/cell/selfstart/querySelfstartInfo.action',pageSize:50,striped:true,toolbar:'#toolbar_tableSelfStartParams',
	                    pagination:true,pagePosition:'bottom',onLoadError:datagridLoadError,onBeforeLoad:tableSelfStartParamsBeforeLoad,onLoadSuccess:datagridLoadSuccess">
            <thead>
              <tr>
                <th data-options="field:'CONFIG_STATUS',fixed:true,formatter:configStatusFormatter" width="50"></th>
                <th data-options="field:'SERIAL_NUMBER'" width="100"><%=rb.getString("XiaoZhanBianMa")%></th>
                <th data-options="field:'CELL_ID'" width="60"><%=rb.getString("XIAOQUID")%></th>
                <th data-options="field:'PCI'" width="60"><%=rb.getString("PCIValue")%></th>
                <th data-options="field:'DL_EARFCN'" width="90"><%=rb.getString("XiaXingPinDian")%></th>
                <th data-options="field:'UL_EARFCN'" width="90"><%=rb.getString("ShangXingPinDian")%></th>
                <th data-options="field:'TAC'" width="60"><%=rb.getString("TAC")%></th>
                <th data-options="field:'PLMN'" width="80"><%=rb.getString("PLMN")%></th>
                <th data-options="field:'PA'" width="60"><%=rb.getString("PA")%></th>
       					<th data-options="field:'PB'" width="60"><%=rb.getString("PB")%></th>
       					<%-- <th data-options="field:'REFERENCESIGPOWER'" width="80"><%=rb.getString("CanKaoXinHaoGongLv")%></th> --%>
                <th data-options="field:'MME_ADDRESS'" width="90"><%=rb.getString("MMEDIZHI")%></th>
                <th data-options="field:'BANDCLASS'" width="80"><%=rb.getString("PinDuan")%></th>
                <th data-options="field:'BANDWIDTH'" width="80"><%=rb.getString("DaiKuan")%></th>
                <th data-options="field:'operation',fixed:true,formatter : selfstartFormatter" width="130"><%=rb.getString("CaoZuo")%></th>
                <%-- <th data-options="field:'PROGRESS_DETAIL'" width="100"><%=rb.getString("ZiPeiZhiJinDu")%></th> --%>
                <%-- <th data-options="field:'FINISH_TIME'" width="100"><%=rb.getString("WanChengShiJian")%></th> --%>
              </tr>
            </thead>
        </table>
	</div>
</div>

<!-- 工具栏  --  配置规划界面  -->
<div id="toolbar_tableSelfStartParams" class="omcTableTool">
	<select id="configStatus" name="type" class="border border-box" style="width:160px;margin-right:1px;height: 26px;vertical-align:bottom;margin-right:20px;">
        <option value=""><%=rb.getString("PeiZhiZhuangTai")%></option>
        <option value="1"><%=rb.getString("PeiZhiChengGong")%></option>
        <option value="2"><%=rb.getString("PeiZhiShiBai")%></option>
        <option value="0"><%=rb.getString("PeiZhiWeiXiaFa")%></option>
	</select>
	<div class="queryGroup">
		<input id="txtSearchSn" placeholder="<%=rb.getString("QingShuRuJiZhanSn")%>" />
		<b onclick="javascript: $('#tableSelfStartParams').datagrid('load');"></b>
	</div>
	<a class="linkbutton" onclick="showConfigRecord()"><span><%=rb.getString("ZiPeiZhiRiZhi")%></span></a>
	<div style="float:right;margin-top:13px;">
		<div class="switch">
			<div isopen="false" class="btnn"></div>
		</div>
		<div class="switchDisabled" style="display:none;"></div>
		<span><%=rb.getString("ZiPeiZhiKaiGuan")%><%=rb.getString("MaoHao")%></span>
	</div>
</div>

<%-- 表单-用于导出自配置页面所有基站信息基站信息 --%>
<form id="export_selfstartParams" style="display:none" method="post"
      action="${ctx}/cell/selfstart/exportSelfstartParamsCsv.action">
</form>

<script>
var ctx = "${ctx}";
var selfstartSwitch = "${SelfstartSwitch}";
var QueRen = "<%=rb.getString("QueRen")%>"
var ShiFouQueDingShanChu = "<%=rb.getString("QueDingShanChuSheBei")%>";
var configStatus = null;

<%--加载完成事件--%>
$(function () {
	<%--清除掉页面完全加载完成之前，所加载的白板--%>
	closeLoading();
	<%--回车事件--%>
	$("#txtSearchSn").bind("keyup", function(e){
		if (e.keyCode == 13){
			$('#tableSelfStartParams').datagrid('load');
		}
	});
	
	//当没有数据时，开关关闭且禁用
	$("#tableSelfStartParams").datagrid({
		onLoadSuccess: function(data){
			if(data.total != 0){
				$(".switchDisabled").hide();
				<%--判断自启动开关应在的样式--%>
				if(selfstartSwitch == "true"){					
					$('.switch').children().attr('isopen','true').animate({left:'33px'});
			        $('.switch').css('background-color','#66CC66');
				}
			}else{
				$(".switchDisabled").show();
				$('.switch').children().attr('isopen','false').animate({left:'2px'});
		        $('.switch').css('background-color','#838383');
			}
		}
	})
		
	
	<%--滑动开关开启关闭事件--%>
    $('.switch').on('click',function(){
		var param = null;
        if ($(this).children().attr('isopen') == 'false') {
            $(this).children().attr('isopen','true').animate({left:'33px'});
            $(this).css('background-color','#66CC66');
            param = {
            	'switch': "1"
            };
        } else {
            $(this).children().attr('isopen','false').animate({left:'2px'});
            $(this).css('background-color','#838383');
            param = {
                'switch': "0"
            };
        }
        $.post("${ctx}/cell/selfstart/setSelfconfigSwitch.action", param, function (data) {
        }, "json");
    });
	
	$("#configStatus").change(function() {
		if ($(this).val() == "1") {
			configStatus = 1;
			$("#tableSelfStartParams").datagrid('reload');
		} else if ($(this).val() == "2") {
			configStatus = 2;
			$("#tableSelfStartParams").datagrid('reload');
		} else if ($(this).val() == "0") {
			configStatus = 0;
			$("#tableSelfStartParams").datagrid('reload');
		} else {
			configStatus = null;
			$("#tableSelfStartParams").datagrid('reload');
		}
	});
})

function tableSelfStartParamsBeforeLoad(param) {
	if(configStatus != null) {
		param["config_status"] = configStatus;
	}
	var searchSn = $("#txtSearchSn").val();
	if(searchSn) {
		param["search_text"] = searchSn;
	}
}

<%-- 右键-小站列表(实时数据) --%>
function showRowMenutableHomeCellList(e, rowIndex, rowData) {
    e.preventDefault();
    if(rowIndex < 0 ){
		return;
	}
    $("#seleNodeBDatagrid").datagrid("clearSelections");
    $("#seleNodeBDatagrid").datagrid("selectRow", rowIndex);
    $("#rowSelfstartList").menu("show", {
        left: e.clientX,
        top: e.clientY
    });
}

function modifyParams() {
	var selCell = $("#seleNodeBDatagrid").datagrid("getSelected");
    if ("Off" == selCell["CONNECTION_STATUS"]) {
        //基站未连接，阻止打开窗口
        $.messager.alert(TiShi, "<%=rb.getString("JiZhanWeiLianJie")%>");
        return;
    }
  
    var url = "${ctx}/cell/selfstart/goSelfstartConfig.action?smallCellCode=" + selCell["SMALL_CELL_CODE"];
    openDefaultWindow(url,{
    	title: '<%=rb.getString("PEIZHILIEBIAO")%>',
    	width: 480,height:500
    });
}

function showMaskWindow() {
	var url = "${ctx}/cell/selfstart/goSelfstartMask.action";
	openDefaultWindow(url,{
		title: '<%=rb.getString("GuiHuaDaoRuDaoChu")%>',
		width: 1000,height:400
	});
}

function selfstartFormatter(value, rowData, rowIndex) {
	var row_id = rowData.SERIAL_NUMBER;
	value = "<div class='grid-edit-btn-div' style='margin-left: 15px' title='<%=rb.getString("XiuGai")%>' onclick='openWinEditTask(\"" + row_id + "\")'></div>" 
	      + "<div class='grid-del-btn-div' style='margin-left:15px;' title='Delete' onclick='deleteDevice(\"" + row_id + "\")'></div>";
	return value;
}

function configStatusFormatter(value, rowData, rowIndex) {
	if ("1" == value) {
		return "<div class='config_success' title='" + "<%=rb.getString("PeiZhiChengGong")%>" + "'><div>";
	} else if ("2" == value) {
		return "<div class='config_failure' title='" + "<%=rb.getString("PeiZhiShiBai")%>" + "'><div>";
	} else {
		return "<div class='config_no' title='" + "<%=rb.getString("PeiZhiWeiXiaFa")%>" + "'><div>";
	}
}

function tableSelfParamsLoadSuccess() {
	$(this).datagrid("fixRownumber");
	
	$(".config_success").jBox('Tooltip',{
		pointer: false,
		color: 'black',
		animation: {open: 'slide:right', close: 'slide:right'}
	});
}

function showConfigRecord() {
	var url = "${ctx}/cell/selfstart/goSelfstartRecord.action";
	openDefaultWindow(url,{
		title: '<%=rb.getString("ZiPeiZhiRiZhi")%>',
		width: 800,height:500
	});
}

function openWinEditTask(idVal) {
	var url = "${ctx}/cell/selfstart/goSelfstartSettings.action?serialNumber="+ idVal;
	openDefaultWindow(url,{
		title: '<%=rb.getString("CanShuPeiZhiXiuGai")%>',
		width: 750,height:520
	});
}

function importSelfstartFile() {
	var url = "${ctx}/cell/selfstart/goSelfstartImportFile.action";
	openDefaultWindow(url,{
		title: '<%=rb.getString("DaoRu")%>',
		width:520,height:250
	});
}

function exportSelfstartFile() {
	var param = {};
	if ($("#configStatus").val()) {
		param.configStatus = $("#configStatus").val();
	}
	if ($("#txtSearchSn").val()) {
		param.txtSearchSn = $("#txtSearchSn").val();
	}
	$("#export_selfstartParams").form({
		"queryParams": param
	}).form("submit",{
		onSubmit: function(param){
			var bool = checkParams(param);
			if(!bool) return false;
        }
	});
}

function deleteDevice(idVal) {
	$("#tableSelfStartParams").datagrid("clearSelections");
	var param = {
		"serialNumber": idVal,
	};
	var msg = ShiFouQueDingShanChu;
	$.messager.confirm(QueRen, msg, function(r) {
    	if (r) {
    		$.post("${ctx}/cell/selfstart/delSelfstartCellinfo.action", param, function(data) {
    			$("#tableSelfStartParams").datagrid("reload");
    		}, "json");
    	} 
    }).addClass("seriousConfirm");
}
</script>