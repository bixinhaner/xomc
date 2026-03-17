<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%@ include file="/common/loading.jsp"%>

<div class="group-title not-extend">
	<span class="title-icon"></span>
	<span class="title-text">VLAN&nbsp;<span id="VlanNum"></span></span>
</div>
<div style='margin-top:25px;margin-left:20px;'>
	<div class="basicInfo  publicParam">
		<label><%=rb.getString("VLANMingCheng")%></label>
		<input type="text" id="vlan_name" class="border-box border" 
		min_length="1" max_length="15"  onblur="validateVlanName(this);"/>
		<span class='operationDiv operation_getFocus'></span>
		<p class='errorTip'><%=rb.getString("ZiMuShuZiXiaHuaXian")%><%=rb.getString("ZiFuChangDu")%>15</p>
	</div>
	<div class="basicInfo publicParam">
		<label>VLAN Id</label>
		<input type="text" id="vlan_id" class="border-box border" min_value="2" max_value="4094" onblur="validateVlanId(this);"/>
		<span class='operationDiv operation_getFocus'></span>
		<p class='errorTip'></p>
	</div>
	<div class='basicInfo'>
		<label><%=rb.getString("XieYiLeiXing")%></label>
		<select id='vlan_proto' class='easyui-combobox border-box border' data-options='editable:false,onSelect:selectVlanProto' style='width:350px;height:26px;'>
			<option value="0">DHCP</option>
			<option value="1">Static</option>
		</select>
		<span class='operationDiv operation_getFocus'></span>
	</div>
	<div class="basicInfo staticParam">
		<label><%=rb.getString("VlanIpDiZhi")%></label>
		<input type="text" id="vlan_ipaddr" class="border-box border" onblur="validateIpVlan(this);"/>
		<span class='operationDiv operation_getFocus'></span>
		<p class='errorTip'><%=rb.getString("IPDiZhi")%></p>
	</div>
	<div class="basicInfo staticParam">
		<label><%=rb.getString("ZiWangYanMa")%></label>
		<input type="text" id="vlan_netmask" class="border-box border" onblur="validateIpVlan(this);"/>
		<span class='operationDiv operation_getFocus'></span>
		<p class='errorTip'><%=rb.getString("IPDiZhi")%></p>
	</div>
	<div class="basicInfo staticParam">
		<label><%=rb.getString("WangGuan")%></label>
		<input type="text" id="vlan_gateway" class="border-box border" onblur="validateIpVlan(this);"/>
		<span class='operationDiv operation_getFocus'></span>
		<p class='errorTip'><%=rb.getString("IPDiZhi")%></p>
	</div>
	<div class="basicInfo staticParam">
		<label>DNS</label>
		<input type="text" id="vlan_dns" class="border-box border" onblur="validateIpVlan(this);"/>
		<span class='operationDiv operation_getFocus'></span>
		<p class='errorTip'><%=rb.getString("IPDiZhi")%></p>
	</div>
</div>
<div class='windowButtonGroup' style='position:absolute;bottom:20px;left:20px;'>
	<a class='linkbutton linkbutton_trend' onclick='saveVlan()'><span><%=rb.getString("QueDing")%></span></a>
	<a class='linkbutton linkbutton_nowanna' onclick='cancelTunnel()'><span><%=rb.getString("QuXiao")%></span></a>
</div>
<script>
	if("${operType}" == "edit" || "${operType}" == "view"){
		var vlanSelect = $("#vlanTableList").datagrid("getSelected");
	}
	$(function(){
		closeLoading();
		setTimeout(function(){
			if("${operType}" == "add"){
				var type = $("#vlan_proto").combobox("getValue");
				if(type == "1"){
					$(".staticParam").show();
				}else{
					$(".staticParam").hide();
				}
				$("#VlanNum").html(vlanList.length+1);
			}
			if("${operType}" == "edit" || "${operType}" == "view"){
				$("#VlanNum").html(vlanSelect.vlan_index);
				$("#vlan_name").val(vlanSelect.vlan_name);
				$("#vlan_name").attr("oldValue",vlanSelect.vlan_name);
				$("#vlan_id").val(vlanSelect.vlan_id);
				$("#vlan_id").attr("oldValue",vlanSelect.vlan_id);
				$("#vlan_proto").combobox("setValue",vlanSelect.proto);
				if(vlanSelect.proto == "1"){
					$(".staticParam").show();
					$("#vlan_ipaddr").val(vlanSelect.ipaddr);
					$("#vlan_netmask").val(vlanSelect.netmask);
					$("#vlan_gateway").val(vlanSelect.gateway);
					$("#vlan_dns").val(vlanSelect.dns);
				}else{
					$(".staticParam").hide();
				}
				if("${operType}" == "view"){
					$("#vlan_name").attr("disabled",true);
					$("#vlan_id").attr("disabled",true);
					$("#vlan_proto").combobox("readonly",true);
					$("#vlan_ipaddr").attr("disabled",true);
					$("#vlan_netmask").attr("disabled",true);
					$("#vlan_gateway").attr("disabled",true);
					$("#vlan_dns").attr("disabled",true);
					$("#addTunnelContent .windowButtonGroup").hide();
				}
			}
		},20)
		
	})
	function validateVlanName(ele){
		var value = $(ele).val();
		var min_length = parseInt($(ele).attr("min_length"));
		var max_length = parseInt($(ele).attr("max_length"));
		var nameStr = [];
		if(vlanList && vlanList.length > 0){
			nameStr = vlanList.map(function(item,index){
				return item.vlan_name;
			})
		}
		if(value.length >= min_length && value.length <= max_length){
			if("${operType}" == "add"){
				if(nameStr.indexOf(value) != -1){
					$(ele).siblings("p.errorTip").show().html("<%=rb.getString("YiCunZai")%>");
					$(ele).addClass("errorBorder");
				}else{
					$(ele).siblings("p.errorTip").hide();
					$(ele).removeClass("errorBorder");
				}
			}else if("${operType}" == "edit"){
				var oldValue = $(ele).attr("oldValue");
				if(value != oldValue && nameStr.indexOf(value) != -1){
					$(ele).siblings("p.errorTip").show().html("<%=rb.getString("YiCunZai")%>");
					$(ele).addClass("errorBorder");
				}else{
					$(ele).siblings("p.errorTip").hide();
					$(ele).removeClass("errorBorder");
				}
			}
		}else{
			$(ele).siblings("p.errorTip").show().html("<%=rb.getString("VLANNameChangDu")%>");
			$(ele).addClass("errorBorder");
		}
	}
	function validateVlanId(ele){
		var value = parseInt($(ele).val());
		var min_value = parseInt($(ele).attr("min_value"));
		var max_value = parseInt($(ele).attr("max_value"));
		var reg = /^\d*$/;
		var idArr = [];
		if(vlanList && vlanList.length > 0){
			idArr = vlanList.map(function(item,index){
				return item.vlan_id;
			})
		}
		if(reg.test(value) && value >= min_value && value <= max_value){
			if("${operType}" == "add"){
				if(idArr.indexOf(value) != -1){
					$(ele).siblings("p.errorTip").show().html("<%=rb.getString("YiCunZai")%>");
					$(ele).addClass("errorBorder");
				}else{
					$(ele).siblings("p.errorTip").hide();
					$(ele).removeClass("errorBorder");
				}
			}else if("${operType}" == "edit"){
				var oldValue = $(ele).attr("oldValue");
				if(value != oldValue && idArr.indexOf(value) != -1){
					$(ele).siblings("p.errorTip").show().html("<%=rb.getString("YiCunZai")%>");
					$(ele).addClass("errorBorder");
				}else{
					$(ele).siblings("p.errorTip").hide();
					$(ele).removeClass("errorBorder");
				}
			}
		}else{
			$(ele).siblings("p.errorTip").show().html("<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%> 2<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%> 4094");
			$(ele).addClass("errorBorder");
		}
	}
	function selectVlanProto(){
		var type = $("#vlan_proto").combobox("getValue");
		if(type == "1"){
			$(".staticParam").show();
		}else{
			$(".staticParam").hide();
		}
	}
	function saveVlan(){
		var vlanFlag = true;
		var vlan_proto = $("#vlan_proto").combobox("getValue");
		if(vlan_proto == "0"){
			$("#addTunnelContent .publicParam input.border").map(function(index,item){
				$(item).blur();
			})
		}else if(vlan_proto == "1"){
			$("#addTunnelContent input.border").map(function(index,item){
				$(item).blur();
			})
		}
		$("#addTunnelContent p.errorTip").map(function(index,item){
			if($(item).is(":visible")){
				vlanFlag = false;
				return;
			}
		})
		if(vlanFlag){
			var vlan_name = $("#vlan_name").val();
			var vlan_id = $("#vlan_id").val();
			var vlan_proto = $("#vlan_proto").combobox("getValue");
			var vlan_ipaddr = $("#vlan_ipaddr").val();
			var vlan_netmask = $("#vlan_netmask").val();
			var vlan_gateway = $("#vlan_gateway").val();
			var vlan_dns = $("#vlan_dns").val();
			var paramObj = {};
			if("${operType}" == "edit"){
				paramObj.vlan_index = vlanSelect.vlan_index;
			}else if("${operType}" == "add"){
				paramObj.vlan_index = vlanList.length + 1;
			}
			paramObj.vlan_name = vlan_name;
			paramObj.vlan_id = vlan_id;
			paramObj.proto = vlan_proto;
			if(vlan_proto == "1"){
				paramObj.ipaddr = vlan_ipaddr;
				paramObj.netmask = vlan_netmask;
				paramObj.gateway = vlan_gateway;
				paramObj.dns = vlan_dns;
			}
			if("${operType}" == "edit"){
				vlanList[vlanSelect.vlan_index-1] = paramObj;
			}else if("${operType}" == "add"){
				vlanList.push(paramObj);
			}
			$("#vlanTableList").datagrid("loadData",vlanList);
			if(vlanList.length == 10){
				$("#addVlanButton").removeClass("titleIcon_add").addClass("titleIcon_add_disabled").removeAttr("onclick");
			}else{
				$("#addVlanButton").removeClass("titleIcon_add_disabled").addClass("titleIcon_add").attr("onclick","addVlan()");
			}
			$("#vlanErrorTip").html("").hide();
			cancelTunnel();
		}
	}
</script>