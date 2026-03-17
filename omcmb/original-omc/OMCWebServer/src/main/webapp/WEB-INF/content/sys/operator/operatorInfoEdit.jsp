<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp"%>

<div class="slidebarTitleDiv">
	<span class="slideTitle"><%=rb.getString("YunYingShang") %></span>&nbsp;&nbsp;<span id='operatorNameModify'></span>
	<div class="slideIcon titleIcon_close" onclick='closeModifyOperator()'></div>
</div>
<div class="slideBody">
	 <div class="omcPageTitleDiv_contain">
		<ul class="omcPageTitleContainer_contain">
			<li class="active"><%=rb.getString("JueSeJiLieBiao")%></li>
		</ul>
	</div>
	<p id='operModifyRoleMes' style='margin-left:55px;margin-top:10px;color:#CC0000;visibility:hidden'><%=rb.getString("QingXuanZeJueSe")%></p>
	<div class="flex-item" style='margin-top:10px;width:760px;margin-left:55px;border:1px solid #CCE1EF;overflow:hidden;'>
		<table  id='operModifyOperatorTable'></table>
	</div>
</div>
<div class="slideFooter">
	<span class="el-button el-button--primary" onclick="saveOperatorModify()"><%=rb.getString("QueDing")%></span>
	<span class="el-button" onclick="closeModifyOperator()"><%=rb.getString("QuXiao")%></span>
</div>
<script>
var operator_code= $('#tableOperatorList').datagrid('getSelections')[0].operator_code;
var operator_name= $('#tableOperatorList').datagrid('getSelections')[0].operator_name;
$(function(){
	closeLoading();
	$('#operatorNameModify').html(operator_name);
	/* 修改运营商中列表 */
	$("#operModifyOperatorTable").datagrid({
		url:'${ctx}/system/operator/getSuperRoleListByOperatorCode.action',
		queryParams : {
			'operator_code':operator_code,
			'type':'modify',
			'role_name':$('#roleNameModify').val()
			},
		singleSelect:false,
		fit:true,
		fitColumns:true,
		border:false,
		rownumbers:true,
		pagePosition:'bottom',
		pagination: true,
		striped: true,
		onLoadSuccess:loadSuccess_modify,
		checkOnSelect:false,
		onCheck:function(){
			$("#operModifyRoleMes").css("visibility","hidden");
		},
		columns: [[
 			{field: 'ck', checkbox: true}, 
			{field: 'role_name',width:100,title:'<%=rb.getString("JueSeMingCheng")%>'}
		]]
	
	
	});
})
/* 关闭修改运营商 */
function closeModifyOperator(){
	 $("#operModifyOperatorDiv").animate({right:"-900px"},350);
	 $("#tableOperatorList").datagrid("reload");
}
function saveOperatorModify(){
	var selectStr = "";
	var selectRows = $('#operModifyOperatorTable').datagrid('getChecked');
	if(selectRows.length == 0){
		$("#operModifyRoleMes").css("visibility","visible");
	}else{
		selectRows.map(function(item,index){
			selectStr += item.ROLE_ID+",";
		})
		selectStr = selectStr.substring(0,selectStr.lastIndexOf(','));
		var params={
				"operator_code":operator_code,
				"role_ids": selectStr
		};
		$.post("${ctx}/system/operator/save.action",params,function(data){
			if(data["success"]){
				showMsg('success_msg','<%=rb.getString("XiuGaiYunYingShangChengGong")%>');
				closeModifyOperator();
			}else{
				showMsg('error_msg',data["message"]);
			}
		},'json')
	}
}
function loadSuccess_modify(data){
	$($("#operModifyOperatorDiv input[type='checkbox']")[0]).css("display","none");
	var rows = data.rows;
	 rows.map(function(item,index){
		if(item.check == "true"){
			$("#operModifyOperatorTable").datagrid('checkRow',index);
		}
		 if(item.ROLE_ID == 1){
			$("#operModifyOperatorDiv input[type='checkbox']")[index+1].disabled = true;
		}
	}) 
}
</script>