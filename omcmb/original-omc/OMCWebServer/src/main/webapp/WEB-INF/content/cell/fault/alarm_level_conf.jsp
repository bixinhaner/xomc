<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<%-- 告警级别配置页面 --%>
<div id="alarmLibInfo" class="easyui-layout" data-options="fit:true">
    <div region="center" data-options="border:false">
    	<div class="easyui-panel" data-options="border:true,fit:true" title="">
	    	<div class="header-title" style="border:none;">
				<h3><%=rb.getString("GaoJingJiBie1")%></h3>
				<ul class="iconText">
					<li><a class="iconExport iconSize" onclick="exportAlarmLib()"><%=rb.getString("DaoChu")%></a></li>
				</ul>
			</div>
		
			<div class="easyui-layout" data-options="border:false,fit:true">
				<div region="north" style="height:68px;overflow: hidden;padding: 20px;" data-options="border:false">
					<input id="txtSearchAlarmLib" style="margin-left: 27px;width:400px;height:26px;float:left;"

							placeholder="<%=rb.getString("QingShuRuGaoJingBiaoShiMaORGaoJingYuanYin")%>" class="border border-box">
					<a class="easyui-linkbutton" onclick="javascript:$('#alarmLibList').datagrid('load');"
							style="margin-left:20px;float:left;"><%=rb.getString("SouSuo")%></a>
					<a class="easyui-linkbutton" onclick="refreshAlarmLib()"
							style="margin-left:20px;float:left;"><%=rb.getString("ShuaXinXinXi")%></a>
				</div>
				<div region="center" data-options="border:false" style="padding:0 20px;">
					<div class="easyui-layout" data-options="fit:true,border:false">
						<div region="center" data-options="border:false">
							<%-- 告警库列表 --%>
							<table class="easyui-datagrid" id="alarmLibList" fit="true" data-options="border:false,fitColumns:true,singleSelect:true,
				                    rownumbers:true,url:'${ctx}/cell/fault/queryAlarmLevelInfosList.action',pageSize:${pageSize},pageList:${pageList},striped:true,
				                    pagination:true,onBeforeLoad:alarmLibListBeforeLoad,pagePosition:'bottom',idField:'ALARM_IDENTIFIER',
				                    onRowContextMenu:showRowMenu_alarmLibList,onLoadError:datagridLoadError,singleSelect:true,onLoadSuccess:datagridLoadSuccess">
								<thead>
								<tr>
									<th data-options="field:'ALARM_IDENTIFIER',sortable:true" width="70"><%=rb.getString("GaoJingWeiYiBiaoZhi")%></th>
									<th data-options="field:'ALARM_NAME',sortable:false" width="120"><%=rb.getString("KeNengYuanYin")%></th>
									<th data-options="field:'SERVERITY_TYPE',sortable:true" width="70"><%=rb.getString("YanZhongChengDu")%></th>
									<th data-options="field:'PROPABLE_CAUSE',sortable:false" width="200"><%=rb.getString("AlarmReason")%></th>
								</tr>
								</thead>
							</table>
						</div>
						<div region="south" data-options="border:true,height:40"></div>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>	
<form id="formExportAlarmLib" style="display:none" method="post" action="">
	<%--已选择的日志查询参数 --%>
	<input type="hidden" name="searchText" />
</form>
<%-- 右键菜单－告警库列表 --%>
<div id="rowMenu_alarmLibList" class="easyui-menu">
      <div>
        <span><%=rb.getString("JiBieChongDingYi")%></span>
        <div style="width: auto">
            <div onclick="resetAlarmServerity('31001','Active')">Critical</div>
            <div onclick="resetAlarmServerity('31002','Active')">Major</div>
            <div onclick="resetAlarmServerity('31003','Active')">Minor</div>
            <div onclick="resetAlarmServerity('31004','Active')">Warning</div>
        </div>
    </div>
</div>

<script type="text/javascript">
$(function(){
	<%-- 首页-告警库信息列表-搜索框回车事件 --%>
	$("#txtSearchAlarmLib").bind("keyup", function(e){
		if (e.keyCode == 13){
			$('#alarmLibList').datagrid('load');
		}
	});
});

// 加载前事件-告警库列表
function alarmLibListBeforeLoad(param) {
	var searchCpeCode = $("#txtSearchAlarmLib").val();
	if (searchCpeCode) {
		param["search_text"] = searchCpeCode;
	}
}

// 显示右键-CPE列表
function showRowMenu_alarmLibList(e, rowIndex, rowData) {
	e.preventDefault();
	if (rowIndex < 0) {
		return;
	}
	$("#alarmLibList").datagrid("clearSelections");
    $("#alarmLibList").datagrid("selectRow", rowIndex);
    
    $("#rowMenu_alarmLibList").menu("show", {
        left: e.clientX,
        top: e.clientY
    });
}

// 打开导出设置窗口
function exportAlarmLib() {
    var searchText = $("#txtSearchAlarmLib").val();
    $("#formExportAlarmLib input[name='searchText']").val(searchText);
	/* $("#formExportAlarmLib").form('submit', {
		url: "${ctx}/cell/fault/exportAlarmLevleResult.action",
		onSubmit: function(param){
			var bool = checkParams(param);
			if(!bool) return false;
		}
	}); */
    exportByForm("${ctx}/cell/fault/exportAlarmLevleResult.action",{
    	searchText: searchText
    });
}

function refreshAlarmLib() {
	$("#alarmLibList").datagrid({
		pageNumber : 1,
		url:'${ctx}/cell/fault/queryAlarmLevelInfosList.action'
	});
}

<%-- 重设告警级别 --%>
function resetAlarmServerity(serverityId, alarmType) {
    var selAlarm = $("#alarmLibList").datagrid("getSelected");
    //如果要重定义的级别跟当前的级别不相同
    var params = {
        alarmIdentifier: selAlarm["ALARM_IDENTIFIER"],
        serverityId: serverityId
    };
    $.post("${ctx}/cell/fault/updateAlarmServerity2.action", params, function (data) {
        if (data["success"]) {
            //修改的告警级别两个列表中可能都有,所以活动告警和历史告警列表都要刷新
            $("#alarmLibList").datagrid("reload");
        } else {
        	showMsg('error_msg',data["message"]);
        }
    }, "json");
}
</script>
