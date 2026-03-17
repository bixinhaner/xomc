<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<style type="text/css">
.textbox.combo .textbox-text {
	padding: 0 4px !important;
}
.textbox.combo{
	vertical-align:top;
}
.titleButtonText {
	transition: opacity 0.5s ease-in;
}
</style>

<div class="panelDefault" style="overflow:hidden">
	<div class="circleIcon" id="sysAddRole"> 
    	<span id='eGWRegistAdd' class="elButton el-icon el-icon-circle-add CODE_EGW hidden" onclick="sysAddRole()"></span>
    	<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
    </div>
	<div class="singleContentDiv" style="top:0;">
		<table id="eGWRegistDatagrid"></table>
	</div>
	<!-- 添加角色下拉框 -->
    <div id="eGWRegistDiv" style="position:absolute;width:100%;;background:#FFFFFF;top:0px;z-index:50;left:0px;bottom:0;display:none;overflow:auto;"></div>
    <!-- 修改角色下拉框 -->
    <div id="eGWModifyDiv" style="position:absolute;width:100%;;background:#FFFFFF;top:0px;z-index:50;left:0px;bottom:0;display:none;overflow:auto;"></div>
</div>
    
<!-- 工具栏-按角色名称查询  -->
 <div id="toolbar_eGWRegistDatagrid" class="toolbarContainer" style="position:relative">
     <div class="queryGroup">
     	<input id='eGWRegistInput'   placeholder="<%=rb.getString("EGWMingCheng")%> / <%=rb.getString("EGWIP")%>">
		<b onclick='$("#eGWRegistDatagrid").datagrid("reload");' class="el-icon el-icon-common-search"></b>
     </div>
</div> 

<script type="text/javascript">

$(function() {
    closeLoading();
	// 表格数据
    $("#eGWRegistDatagrid").datagrid({
		url:'${ctx}/egw/register/queryEgwPageList.action',
		singleSelect:true,
		fit:true,
		fitColumns:true,
		border:false,
		rownumbers:true,
		pagePosition:'bottom',
		pagination: true,
		striped: true,
		toolbar:'#toolbar_eGWRegistDatagrid',
		onLoadSuccess: datagridLoadSuccess,
		onBeforeLoad: beforeload_sysRoleList,
		columns: [[
			{field: 'gw_id',hidden:true},
			{field: 'gw_name',sortable:true,width:100,title:'<%=rb.getString("EGWMingCheng")%>'},
			{field: 'gw_ip',sortable:true,width:100,title:'<%=rb.getString("EGWIP")%>'},
			{field: 'gw_port',sortable:true,width:100,title:'<%=rb.getString("EGWDuanKou")%>'},
			{field: 'operation',fixed:true,width:100,title:'<%=rb.getString("CaoZuo")%>',formatter: sysRoleSetFormatter}
		]]
	});
    $("#eGWRegistInput").bind("keyup", function(e){
		if (e.keyCode == 13){
			$("#eGWRegistDatagrid").datagrid("reload");
			$("#eGWRegistInput").blur();
		}	
	}); 
});
/**
*  表格数据 操作 格式化数据
* @param value{string}    绑定值
* @param rowData{object}   行数据
* @param rowIndex{number}   下标
*/ 
function sysRoleSetFormatter(value, rowData, rowIndex){
	var eGWId = rowData.gw_id;
	var value="";
	value += "<div class='el-icon el-icon-operation-edit CODE_EGW hidden' title='<%=rb.getString("XiuGai")%>' style='margin-left:15px;background-position-x:0;' onclick='modifyeGWInfo(\""+eGWId+"\")'></div>";
	value += "<div class='el-icon el-icon-operation-delete CODE_EGW hidden' title='<%=rb.getString("ShanChu")%>' style='margin-left:15px;background-position-x:0;' onclick='deleteGWInfo(\""+eGWId+"\")'></div>";
	return value;
}

//新增按钮
var showAddDevicePageFlag = true;
function sysAddRole(){
	if(showAddDevicePageFlag){
		
		$(".titleButtonText").html("<%=rb.getString("GuanBi")%>");
		$(".elButton").removeClass("el-icon-circle-add");
		$(".elButton").addClass("el-icon-circle-close");
		$("#eGWRegistDiv").slideDown(500,function(){
			$("#eGWRegistDiv").load("${ctx}/egw/register/toRegisterAdd.action",function(){
				$.parser.parse(this);
			})
		});
		showAddDevicePageFlag = false;
	}else{
		if($('#eGWRegistAdd').hasClass('edit')){
			cancelSysModifyRole();
		}else{
			cancelSysAddRole();
		}	
	}
}

/**
*  修改
* @param eGWId{number}    id
*/ 
function modifyeGWInfo(eGWId){
	$(".titleButtonText").html("<%=rb.getString("GuanBi")%>");
	$(".elButton").removeClass("el-icon-circle-add");
	$(".elButton").addClass("el-icon-circle-close");
	$('#eGWRegistAdd').addClass('edit');
	$("#eGWModifyDiv").slideDown(500,function(){
		$("#eGWModifyDiv").load("${ctx}/egw/register/toRegisterModify.action",function(){
			$.parser.parse(this);
		})
	});
	showAddDevicePageFlag = false;
}
// 关闭新增页面
function cancelSysAddRole(){
	$(".titleButtonText").html("<%=rb.getString("TianJia")%>");
	$(".elButton").addClass("el-icon-circle-add");
	$(".elButton").removeClass("el-icon-circle-close");
	$("#eGWRegistDiv").slideUp(500,function(){
		$("#eGWRegistDiv").html("");
	});
	showAddDevicePageFlag = true;
}
// 关闭修改页面
function cancelSysModifyRole(){
	$(".titleButtonText").html("<%=rb.getString("TianJia")%>");
	$(".elButton").addClass("el-icon-circle-add");
	$(".elButton").removeClass("el-icon-circle-close");
	$('#eGWRegistAdd').removeClass('edit');
	$("#eGWModifyDiv").slideUp(500,function(){
		$("#eGWModifyDiv").html("");
	});
	showAddDevicePageFlag = true;
}


function beforeload_sysRoleList(param){
	param["timeZone"] = timeZone;
	param["searchText"] = $('#eGWRegistInput').val();
}
/**
*  删除
* @param eGWId{number}    id
*/ 
function deleteGWInfo(eGWId){
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingShanChuWangGuan")%>", function (r) {
        if (r) {
        	var params = {
        			"gwId":eGWId
        	}
        	$.post("${ctx}/egw/register/delEgw.action",params,function(data){
                 if (data["success"]) {
                	$("#eGWRegistDatagrid").datagrid("reload");
                } else {
                    showMsg('error_msg',data["message"]);
                    return;
                } 
            }, "json");
        }
    }).addClass("seriousConfirm");
}
</script>