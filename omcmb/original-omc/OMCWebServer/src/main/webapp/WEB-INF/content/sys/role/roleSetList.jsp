<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<style type="text/css">
.textbox.combo .textbox-text {
	padding: 0 4px !important;
}
#form_resetPwd .textbox.combo .textbox-text {
	padding: 4px !important;
}
.omcLogGroup2 .textbox.combo .textbox-text {
	padding: 0 4px !important;
}
.textbox.combo{
	vertical-align:top;
}
.deviceLogContainer{
	width:90%;
	margin-left:60px;
	margin-top:20px;
}
.deviceListContainer,.selectedDevicesList{
	width:45%;
	height:100%;
	
}
.selectButtonContainer{
	width:6%;
	height:100%;
}
.deviceTitlt{
	color:#85A8BF;
	font-size:13px;
}
.flex-ctn{
	display: -webkit-flex;
	display: flex;
	flex-direction: column;
}
.flex-item {
	flex: auto;
	overflow: auto;
}
.titleButtonText {
	transition: opacity 0.5s ease-in;
}
.inputformat label{
	display:block;
	font-size:12px;
	color:#85A8BF;
}
.inputformat input{
	width:400px;
	height:25px;
	border:1px solid #85A8BF;
	display:block;
	margin-top:6px;
	margin-bottom:6px;
}
.inputformat span{
	color:#CC0000;
}
.tree-title{
	cursor:pointer;
}
.tree-icon{
	cursor:pointer;
}
.tree-checkbox:{
	cursor:pointer;
}
#sysFeatureLimitDiv .datagrid-row-over > td {
  background: #fff !important;
  cursor:pointer;
}

#sysViewRoleDiv{
	width:878px;
	position:absolute;
	top:0px;
	bottom:0px;
	background:#fff;
	right:-900px;
	overflow:hidden;
	z-index:100;
}
#syseNBSourceUl .tree-node{
	padding:5px 0px 5px 0px;
}
#sysCPESourceUl .tree-node{
	padding:5px 0px 5px 0px;
}
.circleBg{
	background:none;
}
</style>

<div class="panelDefault" style="overflow:hidden">
	<div class="circleIcon" id="sysAddRole"> 
    	<span id='sysCircleButtonRole' class="circleBg el-icon el-icon-circle-add" onclick="sysAddRole()"></span>
    	<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
    </div>
    <div class='tabsTitle'>
    	<span class='active'>Role Set</span>
    </div>
	<div class="singleContentDiv">
		<table id="sysRoleSetTable"></table>
	</div>
	<!-- 添加角色下拉框 -->
    <div id="sysAddRoleDiv" style="position:absolute;width:100%;;background:#FFFFFF;top:0px;z-index:50;left:0px;bottom:0;display:none;overflow:auto;"></div>
    <!-- 修改角色下拉框 -->
    <div id="sysModifyRoleDiv" style="position:absolute;width:100%;;background:#FFFFFF;top:0px;z-index:50;left:0px;bottom:0;display:none;overflow:auto;"></div>
    <div id='sysViewRoleDiv' class='slidebarPanel'></div>
</div>
    
<!-- 工具栏-按角色名称查询  -->
 <div id="toolbar_sysRole" class="toolbarContainer" style="position:relative">
     <div class="queryGroup">
     	<input id='sysRoleNameInput'   placeholder="<%=rb.getString("JueSeMingCheng")%>">
		<b onclick='$("#sysRoleSetTable").datagrid("reload");' class="el-icon el-icon-common-search"></b>
     </div>
</div> 

<script type="text/javascript">
if($("#curr_operator_span").text() == ""){
	var operator_code = "default";
}else{
	var operator_code = operator_code;
}

$(function() {
    closeLoading();
    $("#sysRoleSetTable").datagrid({
		url:'${ctx}/sys/role/getRoleSet.action',
		queryParams : {
			timeZone:timeZone,
			'operator_code':operator_code,
			'role_name':$('#sysRoleNameInput').val()
			},
		singleSelect:true,
		fit:true,
		fitColumns:true,
		border:false,
		rownumbers:true,
		pagePosition:'bottom',
		pagination: true,
		striped: true,
		toolbar:'#toolbar_sysRole',
		onLoadSuccess:datagridLoadSuccess,
		onBeforeLoad:beforeload_sysRoleList,
		columns: [[
			{field: 'role_name',width:100,title:'<%=rb.getString("JueSeMingCheng")%>'},
			{field: 'upd_user',width:100,title:'<%=rb.getString("QuanXianXiuGaiRen")%>'},
			{field: 'upd_time',width:100,title:'<%=rb.getString("XiuGaiShiJian")%>'},
			{field: 'role_desc',width:100,title:'<%=rb.getString("MiaoShu")%>'},
			{field: 'operation',width:100,title:'<%=rb.getString("CaoZuo")%>',formatter:sysRoleSetFormatter}
		]]
	});
    $("#sysRoleNameInput").bind("keyup", function(e){
		if (e.keyCode == 13){
			$("#sysRoleSetTable").datagrid("reload");
		}	}); 
});

function sysRoleSetFormatter(value, rowData, rowIndex){
	var roleId = rowData.role_id;
	var value="";
	value += "<div class='el-icon el-icon-operation-view' title='<%=rb.getString("ChaKan")%>' style='display:inline-block;cursor:pointer;background-position-x:0;' onclick='openSysViewRoleWindow()'></div>";
	if(rowData.built_in == 9 || rowData.built_in == 8){
		value += "<div class='el-icon el-icon-operation-edit disabled' title='<%=rb.getString("XiuGai") %>' style='margin-left:15px;display:inline-block;cursor:pointer;background-position-x:0;'></div>";
		value += "<div class='el-icon el-icon-operation-delete disabled' title='<%=rb.getString("ShanChu") %>' style='margin-left:15px;display:inline-block;cursor:pointer;background-position-x:0;'></div>";
	}else{
		value += "<div class='el-icon el-icon-operation-edit' title='<%=rb.getString("XiuGai")%>' style='margin-left:15px;display:inline-block;cursor:pointer;background-position-x:0;' onclick='ModifySysRole()'></div>";
		value += "<div class='el-icon el-icon-operation-delete' title='<%=rb.getString("ShanChu")%>' style='margin-left:15px;display:inline-block;cursor:pointer;background-position-x:0;' onclick='deleteSysRoleSet(\""+roleId+"\")'></div>";
	}
	return value;
}

//添加角色
var showAddDevicePageFlag = true;
function sysAddRole(){
	if(showAddDevicePageFlag){
		$(".titleButtonText").html("<%=rb.getString("GuanBi")%>");
		$(".circleBg").removeClass("el-icon-circle-add");
		$(".circleBg").addClass("el-icon-circle-close");
		$("#sysAddRoleDiv").slideDown(500,function(){
			$("#sysAddRoleDiv").load("${ctx}/sys/role/toCreate.action",function(){
				$.parser.parse(this);
				closeLoading();
			})
		});
		showAddDevicePageFlag = false;
	}else{
		if($('#sysCircleButtonRole').hasClass('edit')){
			cancelSysModifyRole();
		}else{
			cancelSysAddRole();
		}	
	}
}

//修改角色
function ModifySysRole(){
	$(".titleButtonText").html("<%=rb.getString("GuanBi")%>");
	$(".circleBg").removeClass("el-icon-circle-add");
	$(".circleBg").addClass("el-icon-circle-close");
	$('#sysCircleButtonRole').addClass('edit');
	$("#sysModifyRoleDiv").slideDown(500,function(){
		$("#sysModifyRoleDiv").load("${ctx}/sys/role/toModify.action",function(){
			$.parser.parse(this);
			closeLoading();
		})
	});
	showAddDevicePageFlag = false;
}

function cancelSysAddRole(){
	$(".titleButtonText").html("<%=rb.getString("TianJia")%>");
	$(".circleBg").addClass("el-icon-circle-add");
	$(".circleBg").removeClass("el-icon-circle-close");
	$("#sysAddRoleDiv").slideUp(500,function(){
		$("#sysAddRoleDiv").html("");
	});
	showAddDevicePageFlag = true;
}

function cancelSysModifyRole(){
	$(".titleButtonText").html("<%=rb.getString("TianJia")%>");
	$(".circleBg").addClass("el-icon-circle-add");
	$(".circleBg").removeClass("el-icon-circle-close");
	$('#sysCircleButtonRole').removeClass('edit');
	$("#sysModifyRoleDiv").slideUp(500,function(){
		$("#sysModifyRoleDiv").html("");
	});
	showAddDevicePageFlag = true;
}

function showTipCircle(ele){
	$(ele).next().css('opacity','1');
}

function hideTipCircle(ele){
	$(ele).next().css('opacity','0');
}

function closeViewSysUserGrooupWindow(){
	$('#sysViewRoleDiv').animate({right:"-900px"},300);
}

function openSysViewRoleWindow(){
	$('#sysViewRoleDiv').animate({right:"0px"},450);
	$('#sysViewRoleDiv').panel({
		href:"${ctx}/sys/role/toView.action",
		width:878
	})
}
function beforeload_sysRoleList(param){
	param["role_name"] = $('#sysRoleNameInput').val();
}

function deleteSysRoleSet(roleId){
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChuJueSe")%>", function (r) {
        if (r) {
        	var params = {
        			"role_id":roleId,
        			"operator_code":operator_code
        	}
        	/* params = JSON.stringify(params); */
        	$.post("${ctx}/sys/role/delete.action",params,function(data){
                 if (data["success"]) {
                	$("#sysRoleSetTable").datagrid("reload");
                } else {
                    showMsg('error_msg',data.message);
                    return;
                } 
            }, "json");
        }
    }).addClass("seriousConfirm");
}

function cancelOtherRoleWindow(){
	if($('#sysCircleButtonRole').hasClass('edit')){
		cancelSysModifyRole();
	}else{
		cancelSysAddRole();
	}	
}
</script>