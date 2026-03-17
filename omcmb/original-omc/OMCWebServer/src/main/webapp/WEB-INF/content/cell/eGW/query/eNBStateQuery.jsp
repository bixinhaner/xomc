<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<script type="text/javascript">
	var ctx = "${ctx}";
</script>

<style type="text/css">
.queryNBInfoItem{
	position:absolute;
	top:113px;
	margin-left:30px;
	z-index:400;
	width:560px;
	height:220px;
	border:1px solid #d1ecf5;
	display:none;
	background:white;
	padding:0 20px;
	box-shadow:0px 10px 20px rgba(88,146,176,0.3);
}
.queryNBInfoItemDiv{
	display:inline-block;
	width:250px;
	margin:8px 40px 0 0;
	vertical-align:top;
}
.queryNBInfoItemDiv label{
	display:block;
	line-height:25px;
	color:#797979;
}
.queryNBInfoItemDiv input{
	width:230px;
}
.queryNBInfoItemDiv img{
	margin-left:10px;
}
.queryNBInfoItemDiv .prompt{
	display:block;
	line-height:25px;
	height:25px;
	color:red;
}
.eNBInfoGroup div{
	display:inline;
}
#winCategoryTree1{
	position:absolute;
	width:900px;
	height:92%;
	background:#FFFFFF;
	box-shadow:2px 3px 16px rgba(158,200,222,0.5);
	border:1px solid #4AB3FF;
	right:-1200px;
	top:8px;
	z-index:200;
	padding:10px 0px 20px 20px;
}
#winCategoryTree1 #treeCellInfoDiv{
	background:#fff;
/* 	height:100%;
	width:100%; */
}
#winCategoryTree1 .header-title{
	padding-right:15px;
}
.eNBStatusItem{
	margin:20px 20px;
	display:flex;
	justify-content:space-between;
	flex-wrap:wrap;
}
.eNBStatusItem > div{
	display:inline-block;
	margin-bottom:25px;
}
.eNBStatusItem > div:nth-child(odd){
	
	margin-right:25px;
}
.eNBStatusItem > div > span{
	display:block;
	width:350px;
}
.newMacroldSetting{
	width:900px;
	height:88%;
	border:1px solid #d1ecf5;
	position:absolute;
	right:-1200px;
	top:8px;
	z-index:200;
	background:white;
	padding:10px 0px 20px 20px;
	box-shadow:0px 10px 20px rgba(88,146,176,0.3);
	overflow:hidden;
}
#eGWAdvanceQuery{
	top:60px;
	display:none;
	position:absolute;
	padding:20px 20px 10px 30px;
	left:0px;
	z-index:100;
	width:100%;
	background:#FFFFFF;
	-webkit-box-shadow:0px 10px 32px rgba(158,200,222,0.35);
}
.highQueryGroup label {
    margin: 0px 8px 0px 0px;    
}
.highQueryGroup li{
	float: left;
    height: 35px;
    margin-right:80px;
    margin-bottom:30px;
    margin-top:10px;
}
.highQueryGroup{
	padding:20px 20px 10px 40px;
	position:relative;
	top:0px;
	box-shadow:0 2px 6px 0 rgba(171,191,221,0) !important;
}
#eGWAdvanceQuery ul li{
	float: left;
    height: 35px;
    margin-right:80px;
    margin-bottom:30px; 
    margin-top:10px;
}
.inputslist label {
    margin: 0px 8px 0px 0px;
}
</style>

<!-- 右上角导出按钮
<div class="omcTitleButton">
	<span class="titleButtonText"><%=rb.getString("DaoChu")%></span><span class="circleBg export_circle" onmouseenter="showTipText(this)" onmouseleave="showTipText(this)" onclick="exportCellLib()"></span>
</div>
 -->
<%-- 表单-用于导出基站列表 --%>
<form id="formExportEGWCellLib" style="display:none" method="post"
      action="${ctx}/eGW/eNBState/exportCellsToCSV.action">
</form>

<%-- 基站状态查询页面 --%>
<div class="panelDefault" style="overflow:hidden;">
	<div class="omcPageTitleDiv">
		<ul class="omcPageTitleContainer">
			<li class="default"><%=rb.getString("JiZhanZhuangTaiChaXun")%></li>
		</ul>
	</div>
	<div class="panelTableDiv">
		<table class="easyui-datagrid" id="tableHomeCellList" fit="true" data-options="border:false,fitColumns:true,singleSelect:true,
               rownumbers:true,url:'${ctx}/eGW/eNBState/getEGWCellInfosPage.action?TimeZone='+timeZone,pageSize:${pageSize},pageList:${pageList},striped:true,
               pagination:true,onBeforeLoad:getParamsBeforeLoad,pagePosition:'bottom',idField:'small_cell_code',toolbar:'#toolbar_tableHomeCellList'">
			<thead>
				<tr>
					<th data-options="field:'SN',sortable:true" width="100"><%=rb.getString("XiaoZhanBianMa")%></th>
					<th data-options="field:'CELL_ID',sortable:true" width="100"><%=rb.getString("JiZhanID")%></th>
					<th data-options="field:'MACRO_ID',sortable:true" width="100"><%=rb.getString("HongZhanID")%></th>
					<th data-options="field:'STATE',formatter:eNBStateFormatter,sortable:true" width="100"><%=rb.getString("ZaiXianZhuangTai")%></th>
					<th data-options="field:'UPDATE_TIME',sortable:true" width="100"><%=rb.getString("ShangXiaShiJian")%></th>
					<th data-options="field:'IP',sortable:true" width="100">IP</th>
					<th data-options="field:'GateWay_IP',sortable:true" width="100"><%=rb.getString("GuiShuWangGuan")%></th>
					<th data-options="field:'operation',formatter: detailFormatter,fixed:true" width="100"><%=rb.getString("CaoZuo")%></th>
				</tr>
			</thead>
		</table>
	</div>
	
	<%-- 查看   --  基站状态查询页面 --%>
	<div id="winCategoryTree1" class="">
		<div class="easyui-layout" data-options="border:false,fit:true">
			<div region="north" data-options="border:false" style="width:880px;height:252px">
				<div style="height: 36px;line-height: 36px;padding: 0 30px 0 20px;">
					<span style="font-size:18px;font-weight:500;color:#2088BF;"><%=rb.getString("JiZhanSheBei")%>：</span>
					<span style="font-size:18px;font-weight:400;color:#2088BF" id="eNBNum"></span>
					<a class="titleIcon_close" onclick="filterClose()" style="float:right;width:36px;height:36px;"></a>
				</div>
			    <div id="treeCellInfoDiv" style="overflow:auto;display:none;"> 
					<div class="eNBStatusItem">
						<div>
							<span for="isEnableLoginPrompt"><%=rb.getString("UEShuLiang")%>：</span>
							<input name="ueNum" id="ueNum" class="easyui-validatebox border border-box item" type="text" style="padding-left: 5px;width:270px;margin-top: 5px;" disabled="disabled">
					 	</div>			
						<div>
							<span for="isEnableLoginPrompt">IP：</span>
							<input name="ueNum" id="eNBIP" class="easyui-validatebox border border-box item" type="text" style="padding-left: 5px;width:270px;margin-top: 5px;" disabled="disabled">
					 	</div>	
						<div>
							<span><%=rb.getString("EGWShangXingLiuLiang")%>：</span>
							<input name="upFlow" id="upFlow" class="easyui-validatebox border border-box item" type="text" style="padding-left: 5px;width:270px;margin-top: 5px;" disabled="disabled">
					 	</div>		
						<div>
						    <span for="sessionExpirationMin"><%=rb.getString("EGWXiaXingLiuLiang")%>：</span>
						    <input name="downFlow" id="downFlow" class="easyui-validatebox border border-box item" type="text" style="padding-left: 5px;width:270px;margin-top: 5px;" disabled="disabled">
				    	</div>			
					</div>	
					<div style="padding:0 20px;">
						<ul class="sysSettings">
							<li>
								<label onclick="openshuntChooseItem()" style="cursor:pointer;">
									<span style="font-size:15px;font-weight:500;"><%=rb.getString("UEXinXi")%>：</span>
									<span class="chooseArrow_UE egw-arrow-down" style="width: 13px;height: 8px;"></span>
								</label>
							</li>
						</ul>
					</div>
				</div>
			</div>
			<div region="center"  data-options="border:false">
				<div style="height:96%;width:95%;margin-top:10px;padding-left:20px;" id="shuntChooseItem">
					<table class="easyui-datagrid" id="ueinfogrid" fit="true" data-options="border:false,fitColumns:true,singleSelect:true,
		                    rownumbers:true,url:'',pageSize:${pageSize},pageList:${pageList},striped:true,
		                    pagination:false,onBeforeLoad:getParamsBeforeLoad,pagePosition:'bottom',idField:'small_cell_code',
		                    onRowContextMenu:''">
						<thead>
							<tr>
								<th data-options="field:'imsi',sortable:false" width="100">Imsi</th>
								<th data-options="field:'onlinetime',sortable:false" width="100"><%=rb.getString("ZaiXianShiJian")%></th>
								<th data-options="field:'upFlow',sortable:false" width="100"><%=rb.getString("EGWShangXingLiuLiang")%></th>
								<th data-options="field:'downFlow',sortable:false" width="100"><%=rb.getString("EGWXiaXingLiuLiang")%></th>
								<th data-options="field:'gwip',sortable:false" hidden=true width="100"><%=rb.getString("EGWIP")%></th>
							</tr>
						</thead>
					</table>
				</div>
				<input type="hidden" id="showtype"/>
			    <input type="hidden" id="GateWay_IP"/>
			    <input type="hidden" id="CELL_ID"/>
			</div>	
	</div>
</div>

<%-- 工具栏  --  基站状态查询页面--%>
<div id="toolbar_tableHomeCellList" class="omcTableTool"> 
 	<form id="eGW_enb_info_query">
 	<div class="easyui-query" name="value" 
 		id="eGwEnbAdvanceQueryDiv" inputId="query_val" targetId="eGWAdvanceQuery"
   		tips="<%=rb.getString("GaoJiChaXun") %>" 
   		placeholder="<%=rb.getString("XiaoZhanBianMa")%>" 
   		data-options="query: queryOperList">
		<a style="margin-left:20px;" href="#" class="linkbutton" onclick="reloadOperList()" ><span><%=rb.getString("ShuaXin")%></span></a>
   	</div>
	    		
     <%-- <div class="defaultQuery" style="padding: 0px 0 10px 10px;z-index:0;">
        <div id="eGwEnbAdvanceQueryDiv" style="vertical-align: bottom;">
            <input name="query_val" id="query_val" placeholder="<%=rb.getString("XiaoZhanBianMa")%>" type="text" style="margin-left:0px;" class="searchInputStyle" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)">
            <div class="highQueryArrow">
                <span class="highQueryTip" id="eGWAdvanceQueryTip" style="cursor:pointer;" onclick="eGWMoreQuery()"><%=rb.getString("GaoJiChaXun")%></span>
                <img class="queryHighBtnErr query-arrow" id="eGWAdvanceQueryImg" onclick="eGWMoreQuery()" flag="1">
            </div>
        </div>
        <div id="kpiAdvanceQueryImg" style="vertical-align: bottom;height: 30px;">
            <b class="searchResultImgChangeStyle" onclick="queryOperList()"></b>
        </div>
		<a style="margin-left:20px;" href="#" class="linkbutton" onclick="reloadOperList()" ><span><%=rb.getString("ShuaXin")%></span></a>
     </div> --%>
     <div id="eGWAdvanceQuery" >         
        	  <ul class="inputslist" style="overflow: hidden;margin-bottom:10px;">
	            <li>
					<label><%=rb.getString("EGWMingCheng")%><%=rb.getString("MaoHao")%></label><br/>
	                <input id="egwName" class="easyui-combobox border border-box combobox-f combo-f textbox-f" style="height: 26px;width:200px;">
	            </li>
	            <li>
	                <label><%=rb.getString("HongZhanID")%><%=rb.getString("MaoHao")%></label><br>
					<input id="macroId" class="easyui-combobox border border-box combobox-f combo-f textbox-f" style="height: 26px;width:200px;">
	            </li>
	            <li>
	                <label ><%=rb.getString("XiaoZhanBianMa")%></label><%=rb.getString("MaoHao")%></label><br>
	                <input id="snumber" maxlength="32" class="easyui-validatebox border border-box" data-options="editable:false" style="height: 26px;width:200px;">
	            </li>
	            <li>
	                <label ><%=rb.getString("JiZhanID")%><%=rb.getString("MaoHao")%></label><br>
	                <input  id="cellId" maxlength="32" class="easyui-validatebox border border-box" data-options="editable:false" style="height: 26px;width:200px;">
	            </li>
	        </ul>
	        <div class="linkbuttonGroup" style="margin-bottom:20px">
	        	<a href="#" class="linkbutton linkbutton_trend" onclick="queryOperList()"><span><%=rb.getString("ChaXun")%></span></a>
	        	<a href="#" class="linkbutton linkbutton_nowanna" onclick="reseteGWAdvanceQuery()"><span><%=rb.getString("ChaXunChongZhi")%></span></a>
	        </div>
        </div> 
    </form>    
</div> 
<%-- 窗口-进度条 --%>
<div id="winLoadingPro" title="<%=rb.getString("ShuJuQingQiu")%>" class="easyui-window"
		data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:400,height:200,resizable:false,closable:false">
    <span class="loading-gif" style="margin-left:10px;margin-top:10px;"></span>
    <span style="margin-left:75px;"><%=rb.getString("ShuJuQingQiu")%></span>
</div>

<script type="text/javascript">
	$(function() {
		getSelecteGW();
		getSelectMacroId();
		//高级查询
    	/* $("#query_val").focus(function(){
    		$("#eGWAdvanceQueryTip").show();
    	});
    	$("#eGWAdvanceQueryImg").mouseenter(function(){
    		if($("#query_val").is(":focus")){
    			return;
    		}
    		$("#eGWAdvanceQueryTip").show();
    	});
    	$("#eGWAdvanceQueryImg").mouseleave(function(){
    		if($("#query_val").is(":focus")){
    			return;
    		}
    		if($("#eGWAdvanceQuery").is(":visible")){
    			return;
    		}
    		$("#eGWAdvanceQueryTip").hide();
    	});
    	$("#eGWAdvanceQueryTip").mouseleave(function(){
    		if($("#query_val").is(":focus")){
    			return;
    		}
    		if($("#eGWAdvanceQuery").is(":visible")){
    			return;
    		}
    		$("#eGWAdvanceQueryTip").hide();
    	}); */
    	/* $(document).click(function(e){
    		if($(e.target).closest(".window-mask").length==0
    			 &&$(e.target).closest(".messager-window").length==0&&$(e.target).closest(".combo-p").length==0
    			 &&$(e.target).closest("#eGWAdvanceQuery").length==0&&$(e.target).closest("#eGWAdvanceQueryTip").length==0
    			 &&$(e.target).closest("#eGWAdvanceQueryImg").length==0&&$(e.target).closest("#query_val").length==0
    			 &&$(e.target).closest(".searchResultImgChangeStyle").length==0
    			 &&$(e.target).closest(".calendar-other-month").length==0){
    			if($("#eGWAdvanceQuery").is(":visible")){	
    					$("#eGWAdvanceQuery").slideUp(400);	
    				}else{					
    					$("#eGWAdvanceQuery").slideUp();	
    				}	
    				$("#eGWAdvanceQueryImg").removeClass('expanded');
    				$("#eGWAdvanceQueryImg").attr("flag","1"); 
    				$("#eGWAdvanceQueryTip").hide();
    		}
    	}); */
	});
	
	function eGWMoreQuery(){
		if($("#eGWAdvanceQueryImg").attr("flag")=="1"){
			$("#eGWAdvanceQuery").slideDown(500);
			$("#eGWAdvanceQueryImg").attr("flag","0");
			$("#eGWAdvanceQueryImg").addClass('expanded');
		}else{
			$("#eGWAdvanceQuery").slideUp(500);
			$("#eGWAdvanceQueryImg").attr("flag","1");
			$("#eGWAdvanceQueryImg").removeClass('expanded');
		}
	}
	function reseteGWAdvanceQuery(){	
		$("#query_val").val("");
		$("#egwName").combobox('setValue', '');
		$("#macroId").combobox('setValue', '');  	
		$("#snumber").val("");
		$("#cellId").val("");
	}
	function reloadOperList(){
		$("#tableHomeCellList").datagrid('reload');
	}
	
	function cancelQueryeNB(){
		$(".queryNBInfoItem").slideUp(300);
	}
	
	function detailFormatter(value, rowData, rowIndex){
		var res = "";
		if(rowData.STATE=="Active"){
			res = "<div class='operationDiv operation_view' style='margin-left: 15px' title='<%=rb.getString("ChaKan")%>' onclick='showCategoryTree(true,\"" + rowData.SN + "\",\""+rowData.CELL_ID+"\",\""+rowData.GateWay_IP+"\")'></div>";
		}
		return res;
	}
	
	function queryOperList(){
		$(".queryNBInfoItem").slideUp(300);
		var query_val = $("input[name='query_val']").val();
		var snumber = $("#snumber").val();
		var cellId = $("#cellId").val();
		if(query_val==""){
			query_val=snumber;
		}
		
		var egwName = $("#egwName").combobox('getValue');
		var macroId = $("#macroId").combobox('getValue');
		var param = {
                SN : query_val,
                gwIp:egwName,
                macroId:macroId,
                cellId:cellId
       		}
       $("#tableHomeCellList").datagrid('load', param);
	   <%-- $("#snumber,#cellId").val("");
	   $("#egwName,#macroId").combobox({
			valueField:'value',
			textField:'text',
			data:[{text:"<%=rb.getString("QuanXuan")%>",value:""}]
	   }); --%>
	   $("#eGWAdvanceQuery").slideUp(500);
	   $("#eGWAdvanceQueryImg").attr("flag","1");
	   $("#eGWAdvanceQueryImg").removeClass('expanded');
	}
	
	function getParamsBeforeLoad(param){
		//var imsi_val = $("input[name='imsi_val']").val();
		//	param["SN"] = imsi_val;
	}
	
	function getSelecteGW(){
		$("#egwName").empty();
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
		
		$("#egwName").combobox({
			valueField:'value',
			textField:'text',
			data:GW_INIT,
			onSelect:function(obj){
				if(obj.value!=""){
					getSelectMacroId(obj.value);
				}else{
					$("#macroId").empty();
					var Macroid_INIT=[{text:"<%=rb.getString("QuanXuan")%>",value:""}];
					$("#macroId").combobox({
						valueField:'value',
						textField:'text',
						data:Macroid_INIT
					});
				}
			}
		});
	}
	
	function getSelectMacroId(gwip){
		var GW_INIT=[{text:"<%=rb.getString("QuanXuan")%>",value:""}];
		var params = {gwIp: gwip};
		$.ajax({
			type: "post",
			url: "${ctx}/eGW/eNBState/getMacroIdBygwIp.action?TimeZone="+timeZone,
			async: false,
			data:params,
			dataType:"json",
			success: function(data) {
				$.each(data,function(idx,obj){
					 var macroId ={text:obj.MACRO_ID,value:obj.MACRO_ID};
					 GW_INIT.push(macroId);
				 });
			}
		});
		
		$("#macroId").combobox({
			valueField:'value',
			textField:'text',
			data:GW_INIT
		});
	}
	
	function eNBStateFormatter(value, rowData, rowIndex){
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
		
	function showCategoryTree(open,SN,CELL_ID,GateWay_IP) {
		$("#ueNum,#upFlow,#downFlow,#eNBIP").val("");
		if (open) {
			//$("#ueinfogrid").datagrid('reload');
			//查询小站信息
			var params = {cellid: CELL_ID,GateWay_IP:GateWay_IP};
			$("#winLoadingPro").window("open");
			//定时器判断请求超时
			var num=0;
			var loadingtime = setInterval(function(){
				num++;
				if(num>6){
					clearInterval(loadingtime);
					$("#winLoadingPro").window("close");
					$.messager.alert(TiShi, "request timeout.");
				}
			},1000);
			
	 		$.ajax({
				type: "post",
				url: "${ctx}/eGW/eNBState/getSelCell.action?TimeZone="+timeZone, 
				data: params,
				async: true,
				dataType:"json",
				success: function(data) {
					clearInterval(loadingtime);
					$("#winLoadingPro").window("close");
					if(data==undefined||data==""||data==null){
						$.messager.alert(TiShi, "<%=rb.getString("QingQiuYiChang")%>");
						$("#tableHomeCellList").datagrid('reload');
						return false;
					}
					$("#shuntChooseItem").hide();	
					$("#showtype").val(0);
					$(".chooseArrow_UE").removeClass('egw-arrow-up').addClass('egw-arrow-down');
					$("#ueNum,#upFlow,#downFlow,#eNBIP").val("");
					$("#ueinfogrid").datagrid("loadData",{rows:[]});
					$("#eNBNum").text(SN);
					$(window).resize(function(){
					})
					
					$("#winCategoryTree1").animate({right:'0px'},500);  
					$("#treeCellInfoDiv").show();
					$("#CELL_ID").val(CELL_ID);
					$("#GateWay_IP").val(GateWay_IP);
					$("#ueNum").val(data.ueAccessNum);
					$("#upFlow").val(data.upFlow);
					$("#downFlow").val(data.downFlow);
					$("#eNBIP").val(data.ip);
				},
				error:function(xmlhttprequest,textstatus,errorThrown){
					clearInterval(loadingtime);
					$("#winLoadingPro").window("close");
					if(textstatus=="timeout"){
						$.messager.alert(TiShi, "<%=rb.getString("QingQiuChaoShi")%>");
					}else{
						$.messager.alert(TiShi, "<%=rb.getString("QingQiuYiChang")%>");
					}
				}
	 		});	
			//$.post("${ctx}/eGW/eNBState/getSelCell.action", params, function(data){}, "json");
		}
	}
	
	function filterClose(){
		$("#treeCellInfoDiv").hide();
		$("#winCategoryTree1").animate({right:'-1280px'},350);
	}
	
	//展开UE在线用户信息
	function openshuntChooseItem(){
		var showtype = $("#showtype").val();
		if(showtype == 0){
			$("#shuntChooseItem").show();
			$("#showtype").val(1);
			$(".chooseArrow_UE").removeClass('egw-arrow-down').addClass('egw-arrow-up');
			//查询小站信息
			var params = {cellid: $("#CELL_ID").val(),GateWay_IP:$("#GateWay_IP").val()};
			$.post("${ctx}/eGW/eNBState/getSelCell.action", params, function(data){
				$("#ueNum").val(data.ueAccessNum);
				$("#upFlow").val(data.upFlow);
				$("#downFlow").val(data.downFlow);
				$("#eNBIP").val(data.ip);
			}, "json");
			
			setTimeout(function(){
				var cellid = $("#CELL_ID").val();
				var GateWay_IP = $("#GateWay_IP").val();
				$("#ueinfogrid").datagrid({
			    	   url:'${ctx}/eGW/eNBState/getUEByCell.action?TimeZone='+timeZone,
			    	   pagination:false,
			           queryParams:{
			        	   cellid:cellid,
			        	   GateWay_IP:GateWay_IP,
			        	   TimeZone:timeZone
			           }
			        });
			},300);
		}else{
			$(".chooseArrow_UE").removeClass('egw-arrow-up').addClass('egw-arrow-down');
			$("#shuntChooseItem").hide();
			$("#showtype").val(0);	
		}
		//$("#ueinfogrid").datagrid('insertRow',{index:1,row:{imsi:1,onlinetime:2,upFlow:3,downFlow:4,gwip:5}});
	}
	
	// 打开导出设置窗口
	function exportCellLib() {
	    var searchText = $("#txtSearchCellLib").val();
	    $("#formExportEGWCellLib input[name='searchText']").val(searchText);
		/* $("#formExportEGWCellLib").form('submit', {
			url: "${ctx}/cell/fault/exportAlarmLevelResult.action",
			onSubmit: function(param){
				var bool = checkParams(param);
				if(!bool) return false;
			}
		}); */
	    exportByForm("${ctx}/cell/fault/exportAlarmLevelResult.action",{
	    	searchText: searchText
	    });
	}
</script>