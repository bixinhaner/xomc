<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<script type="text/javascript">
	var ctx = "${ctx}";
</script>

<%-- UE信息查询页面 --%>
<div class="panelDefault">
	<div class="omcPageTitleDiv">
		<ul class="omcPageTitleContainer">
			<li class="default"><%=rb.getString("UEXinXiChaXun")%></li>
		</ul>
	</div>
	<div class="panelTableDiv">
		<table class="easyui-datagrid" id="tableHomeCellList" fit="true" data-options="border:false,fitColumns:true,singleSelect:true,
               rownumbers:true,url:'',striped:true,pagination:false,onBeforeLoad:getParamsBeforeLoad,idField:'small_cell_code',toolbar:'#toolbar_tableHomeCellList'">
			<thead>
				<tr>
					<th data-options="field:'imsi',sortable:false" width="100">Imsi</th>
					<th data-options="field:'gateway',sortable:false,hidden:true" width="150"><%=rb.getString("GuiShuWangGuan")%></th>
					<th data-options="field:'cellid',sortable:false,hidden:true" width="100"><%=rb.getString("JiZhanID")%></th>
					<th data-options="field:'onlinetime',sortable:false" width="100"><%=rb.getString("ZaiXianShiJian")%></th>
					<th data-options="field:'upFlow',sortable:false" width="100"><%=rb.getString("EGWShangXingLiuLiang")%></th>
					<th data-options="field:'downFlow',sortable:false" width="100"><%=rb.getString("EGWXiaXingLiuLiang")%></th>
				</tr>
			</thead>
		</table>
	</div>
</div>

<%-- 工具栏  -- UE信息查询页面 --%>
<div id="toolbar_tableHomeCellList" class="omcTableTool">
	<div style="display:inline-block;margin-right:25px;">
		<label style="width: 50px; display: inline-block" class="borderBoxClass"><%=rb.getString("eGW")%>：</label>
		<select id="S_GateWay" class="easyui-combobox border border-box"  style="height:26px;width:230px;"></select>
	</div>
	<div class="queryGroup">
		<input name="imsi_val" id="imsi_val" placeholder="Imsi" maxlength="15" onkeyup="value=value.replace(/[^\d]/g,'')"  type="text" />
		<b onclick="queryOperList()"></b>
	</div>
</div>

<script type="text/javascript">
	$(function() {		
		getSelecteGW();
	});
	
	function getSelecteGW(){
		$("#S_GateWay").empty();
		var GW_INIT=[{text:"<%=rb.getString("QuanXuan")%>",value:""}];
		$.ajax({
			type: "post",
			url: "${ctx}/eGW/eNBState/getSelecteGWList.action?TimeZone="+timeZone,
			async: false,
			dataType:"json",
			success: function(data) {
				$.each(data,function(idx,obj){
					 var gw_ip ={text:obj.GW_NAME+"["+obj.GW_IP+"]",value:obj.GW_IP};
					 GW_INIT.push(gw_ip);
				});
			}
		});
		
		$("#S_GateWay").combobox({
			valueField:'value',
			textField:'text',
			data:GW_INIT,
			onSelect:function(obj){

			}
		});
	}
	
	function queryOperList(){
		var imsi_val = $("input[name='imsi_val']").val();
		if(imsi_val==""){
			$.messager.alert(TiShi, "Imsi cannot be null.");
			$("#imsi_val").focus().select();
			return false;
		}
		
		if(imsi_val.length!=15){
			$.messager.alert(TiShi, "Length is not enough.");
			$("#imsi_val").focus().select();
			return false;
		}
		
		var S_GateWay = $("#S_GateWay").combobox('getValue');
        $("#tableHomeCellList").datagrid({
    	   url:'${ctx}/eGW/ueinfo/egwQuryRegByImsi.action?GateWay_IP=&TimeZone='+timeZone,  
    	   queryParams:{
    		   imsi : imsi_val,
               gate_way: S_GateWay,
               TimeZone: timeZone
    	   }
        });
	}
	
	function getParamsBeforeLoad(param){
		
	}
	
	function ueStateFormatter(value, rowData, rowIndex){
		if (value == null) {
			return null;
		}
		else if (value == "Active") {
			value = "<%=rb.getString("ZaiXian")%>";
		}
		else{
			value = "<%=rb.getString("LiXian")%>";
		}
		return value;
	}
	
</script>