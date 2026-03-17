<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%@ include file="/common/loading.jsp"%>
<div class="group-title not-extend">
	<span class="title-icon"></span>
	<span class="title-text">Route&nbsp;<span id="routeNum"></span></span>
</div>
<div style='margin-top:25px;margin-left:20px;'>
	<div class="basicInfo">
		<label><%=rb.getString("MuBiaoIP")%></label>
		<input type="text" id="route_targetIp" class="border-box border" onblur="validateIpVlan(this);"/>
		<span class='operationDiv operation_getFocus'></span>
		<p class='errorTip'><%=rb.getString("IPDiZhi")%></p>
	</div>
	<div class="basicInfo">
		<label><%=rb.getString("WangGuan")%></label>
		<input type="text" id="route_gateway" class="border-box border" onblur="validateIpVlan(this);"/>
		<span class='operationDiv operation_getFocus'></span>
		<p class='errorTip'><%=rb.getString("IPDiZhi")%></p>
	</div>
	<div class="basicInfo">
		<label><%=rb.getString("ZiWangYanMa")%></label>
		<input type="text" id="route_netmask" class="border-box border" onblur="validateIpVlan(this);"/>
		<span class='operationDiv operation_getFocus'></span>
		<p class='errorTip'><%=rb.getString("IPDiZhi")%></p>
	</div>
</div>
<div class='windowButtonGroup' style='position:absolute;bottom:20px;left:20px;'>
	<a class='linkbutton linkbutton_trend' onclick='saveRoute()'><span><%=rb.getString("QueDing")%></span></a>
	<a class='linkbutton linkbutton_nowanna' onclick='cancelTunnel()'><span><%=rb.getString("QuXiao")%></span></a>
</div>
<script>
	if("${operType}" == "edit" || "${operType}" == "view"){
		var routeSelect = $("#routeTableList").datagrid("getSelected");
	}
	$(function(){
		closeLoading();
		if("${operType}" == "add"){
			$("#routeNum").html(routeList.length+1);
		}
		if("${operType}" == "edit" || "${operType}" == "view"){
			$("#routeNum").html(routeSelect.route_index);
			$("#route_targetIp").val(routeSelect.target);
			$("#route_gateway").val(routeSelect.gateway);
			$("#route_netmask").val(routeSelect.netmask);
			if("${operType}" == "view"){
				$("#route_targetIp").attr("disabled",true);
				$("#route_gateway").attr("disabled",true);
				$("#route_netmask").attr("disabled",true);
				$("#addTunnelContent .windowButtonGroup").hide();
			}
		}
		
	})
	function saveRoute(){
		var routeFlag = true;
		$("#addTunnelContent input.border").map(function(index,item){
			$(item).blur();
		})
		$("#addTunnelContent p.errorTip").map(function(index,item){
			if($(item).is(":visible")){
				routeFlag = false;
				return;
			}
		})
		if(routeFlag){
			var route_targetIp = $("#route_targetIp").val();
			var route_gateway = $("#route_gateway").val();
			var route_netmask = $("#route_netmask").val();
			var paramObj = {};
			if("${operType}" == "edit"){
				paramObj.route_index = routeSelect.route_index;
			}else if("${operType}" == "add"){
				paramObj.route_index = routeList.length + 1;
			}
			paramObj.target = route_targetIp;
			paramObj.gateway = route_gateway;
			paramObj.netmask = route_netmask;
			if("${operType}" == "edit"){
				routeList[routeSelect.route_index-1] = paramObj;
			}else if("${operType}" == "add"){
				routeList.push(paramObj);
			}
			$("#routeTableList").datagrid("loadData",routeList);
			$("#routeErrorTip").html("").hide();
			cancelTunnel();
		}
	}
</script>