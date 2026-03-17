<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<%-- 任务管理界面 --%>

<div class="easyui-layout" data-options="border:false,fit:true">
	<div region="west" data-options="border:false,split:false,width:200" style="padding-right:15px;background-color: #F3F3F4;">
	    <div class="easyui-panel" data-options="border:true, fit:true" title="<%=rb.getString("RenWu")%>">
		    <ul id="taskTypeTree"></ul>
		</div>
	</div>
	<div region="center" id="centerPanel" data-options="border:false,href:'${ctx}/task/MMLScript/toMMLScriptTaskList.action'">
	</div>
</div>

<script type="text/javascript">
var operation_ShanChu = "${role_operation_shanchu}";
var operation_ChongQi = "${role_operation_ChongQi}";
var operation_FactoryReset = "${role_operation_FactoryReset}";
var oparation_role_list = [operation_ShanChu, operation_ChongQi, operation_FactoryReset];

var taskTypeTreeData = [
{
	"text": "<%=rb.getString("MMLJiaoBen")%>",
	"url": "/task/MMLScript/toMMLScriptTaskList.action"
}
];

$(function() {
	/* if (operation_ChongQi != "true") {
		taskTypeTreeData.splice(0,1);
	}
	if (operation_FactoryReset != "true") {
		taskTypeTreeData.splice(0,1);
	} */ 
	createSecondMenu(taskTypeTreeData, typeTreeOnClick, $("#taskTypeTree"));
});

// 任务类型树，节点点击事件
function typeTreeOnClick(node) {
	$("#centerPanel").panel({
		href: "${ctx}" + node.attr("url")
	});
}

//任务状态格式化：激活/挂起
function taskStatusFmt(value, rowData, rowIndex) {
	if (value == "0") {
		return "<%=rb.getString("JiHuo")%>";
	} else if (value == "1") {
		return "<%=rb.getString("GuaQi")%>";
	}
}

// 任务执行结果格式化
function taskResultFmt(value, rowData, rowIndex) {
	if (value == "0") {
		return "<%=rb.getString("ChengGong")%>";
	} else if (value == "1") {
		return "<%=rb.getString("BuFenChengGong")%>";
	} else if (value == "2") {
		return "<%=rb.getString("ShiBai")%>";
	} else if (value == "3") {
		return "<%=rb.getString("ZhongZhi")%>";
	} else {
		return "";
	}
}

// 任务进度格式化
function taskProgressFmt(value, rowData, rowIndex) {
	if (value == "0") {
		return "<%=rb.getString("WeiKaiShi")%>";
	} else if (value == "1") {
		return "<%=rb.getString("JinXingZhong")%>";
	} else if (value == "2") {
		return "<%=rb.getString("YiJieShu")%>";
	}
}
</script>