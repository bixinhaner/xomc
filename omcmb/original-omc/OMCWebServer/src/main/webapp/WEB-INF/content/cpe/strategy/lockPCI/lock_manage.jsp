<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<style type="text/css">
.question-mark {
	background-image: url(${ctx}/css/images/global/question_mark.png);
	background-position: center center;
	display: inline-block;
	width: 40px;
	height: 40px;
	vertical-align: top;
	background-repeat: no-repeat;
}
#toolbar_PCITaskList .textbox.combo{
	vertical-align:top;
}

#CPEPCIQueryDiv li{
	float: left;
    height: 35px;
    margin-right:80px;
    margin-bottom:30px;
    margin-top:10px;
}
.inputslist label {
    margin: 0px 8px 0px 0px;
}
.textbox.combo .textbox-text {
	padding: 0 4px !important;
}
.showOp{
    right:0px;
	left:1px;
}
/*new pci lock*/
.newPCIContainer{
	display:none;
	/* width:97%;
	height:94%; */
	position:absolute;
	z-index:11;
	top:46px;
	right:20px;
	left:20px;
	bottom:0px; 
	background:white;
	overflow:auto;
	/* padding:30px 0px 50px 75px; */
}
.pciBoxcontainer{
	min-height:160px;
}
.baseInfoBox{
	min-height:50px;
	margin-top:20px;
}
.baseInfoBox li{
    width:400px;
    height:55px;
	float:left;
	margin-left:40px;
}
.baseInfoBox li span{
	display:block;
	height:20px;
	line-height:20px;
	color:#85A8BF;
}
.baseInfoBox li input{
	padding-left:15px;
	height:26px;
	line-height:26px;
	border:1px solid #85A8BF;
	width:380px;
}
.executedMode p{
	margin-left:40px;
	font-size:13px;
	height:40px;
	line-height:50px;
	color:#CC0000;
}
.execModeBox input{
	margin-right:7px;
	vertical-align:middle;
	margin-top:-1px;
	margin-bottom:2px;
}
.execModeBox .labelBox{
	height:32px;
	padding-top:18px;
	margin-left:40px;
}
.execModeBox .labelBox .textbox-text{
	line-height:26px;
}
.execModeBox label{
	display:inline-block;
	height:14px;
	line-height:14px;
	font-size:12px;
	color:#85A8BF;
}
.eNBTitle{
	height:45px;
	line-height:45px;
	color:#4AC3FF;
	margin-bottom:5px;
	margin-left:40px;
}
.selected_device_box{
	width：70%;
	min-height:275px;
}
.selected_device_box .selected_device_title{
	margin-left:40px;
	color:#85A8BF;
}
.selected_device_box .selected_device_container{
	width:990px;
	height:162px;
	border:1px solid #CAE4E7;
	margin-left:42px;
	margin-top:6px;
	padding-left:12px;
	padding-top:12px;
	padding-bottom:10px;
	position:relative;
	overflow:auto;
}
.repeatAlermTitle{
	width:100%;
	height:20px;
	margin-left:40px;
	line-height:20px;
	color:red;
	display:none;
}
.selectListItem{
	width:95%;
	height:30px;
	margin-top:7px;
	display:none;
}
.selectListItem .delList{
	display:inline-block;
	float:left;
	width:20px;
	height:20px;
	margin-top:5px;
	background:url(${ctx}/css/images/newIcon/titleIcon/titleIcon_selection_del.png)
}
.selectListItem .delList:hover{
	background:url(${ctx}/css/images/newIcon/titleIcon/titleIcon_selection_del_hover.png)
}
.selectListItem .cellNameList{
	display:inline-block;
	float:left;
	width:235px;
	height:30px;
	line-height:30px;
	fone-size:12px;
	color:#949494;
	margin-left:18px;
	margin-right:60px;
}
.FreList,.pciList,.changeCpeList{
	display:inline-block;
	height:30px;
	line-height:30px;
	color:#85A8BF;
	float:left;
}
.inpList{
	height:28px;
	width:100px;
	padding:0px 6px;
	border:1px solid #1DA3FC;
	line-height:28px;
	float:left;
	margin-right:30px;
}
#PCICPEPCILockBord
	
}
#PciLockDiv .datagrid-header-check input{
	display:none;
}
.lockAlarm{
	position:absolute;
	background:white;
	width:144px;
	/*height:40px; */
	padding:5px 5px;
	box-shadow:4px 4px 19px 0 rgba(200,211,231,0.7);
	border:1px solid #d1ecf5;
	left:0;
	top:30px;
	z-index:99999;
	display:none;
}
.lockAlarm p{
	height:20x;
	line-height:20px;
	font-size:14px;
	color:red;
}
.disabledConfirm + div > a:first-of-type > span{
	background:red;
}
.suoPinInputContainer{
	height:100px;
	width:850px;
	margin-top:30px;
	margin-left:40px;
}
.suoPinInputContainer .pcilocksignInput{
	float:left;
	width:400px;
	height:80px;
}
.suoPinInputContainer .pcilocksignInput p{
	height:30px;
	line-height:30px;
	width:400px;
	font-size:13px;
	color:#85A8bF;
}
.suoPinInputContainer .pcilocksignInput input{
	display:inline-block;
	height:28px;
	width:398px;
	border:1px solid #85A8bF;	
	outline:none;
}
.suoPinInputContainer .pcilocksignInput span{
	display：inline-block;
	height:20px;
	line-height:20px;
	color:#CC0000;
	font-size:13px;
	display:none;
}
.successTotask{
	float:left;
	min-width:200px;
	height:38px;
	line-height:38px;
	text-indent:50px;
	/* text-align:center; */
	margin:30px 0 0 35px;
	color:#508D9B;
	font-size:16px;
	font-weight:bold;
	background:#E9FBFF url("${ctx}/images/success.png") no-repeat 16px center;
	
}
</style>

<!-- 右上角添加按钮 -->
<div class="omcTitleButton CODE_CPE_PCI_LOCK hidden">
	<span class="titleButtonText"><%=rb.getString("TianJia")%></span><span class="circleBg add_circle" onmouseenter="showTipText(this)" onmouseleave="showTipText(this)" onclick="newcpePCILockTask()"></span>
</div>


<%-- PCILOCK --%>
<div class="panelDefault">
	<div class="omcPageTitleDiv">
		<ul class="omcPageTitleContainer">
			<li class="default" id="cpepciTitle"><%=rb.getString("PinDianSuo")%></li>
		</ul>
	</div>
	<!-- 表格 -->
	<div class="panelTableDiv" id="cpePCItaskTable">		
		<table class="easyui-datagrid"  id="CPEPCITaskList" fit=true
				data-options="border:false,fit:true,fitColumns:true,singleSelect:true,rownumbers:true,striped:true,pagination:true,pagePosition:'bottom',idField:'TASK_ID',
				toolbar:'#toolbar_PCITaskList',queryParams:{timeZone:timeZone},
				url:'${ctx}/cpe/strategy/getPciLockTaskList.action',onBeforeLoad:tablePciLockTaskListBeforeLoad,onLoadSuccess:pci_lock_cpe_grid_load_success,onLoadError:datagridLoadError">
            <thead>
	            <tr>
	             	<th data-options="field:'operation',formatter:pciLockTaskFormatter,styler:setStyle,fixed:true" width="80"><%=rb.getString("CaoZuo")%></th>
	             	<th data-options="field:'TASK_NAME',sortable:true" width="200"><%=rb.getString("RenWuMingCheng")%></th>
	             	<th data-options="field:'LOCK_MODE',formatter:frequencyFormatter,sortable:true" width="200"><%=rb.getString("SaoMiaoFangShi")%></th>
	             	<th data-options="field:'TASK_PROGRESS',formatter:taskProgressFmt,sortable:true" width="120"><%=rb.getString("ZhuangTai")%></th>
	             	<th data-options="field:'PERCENT_PROGRESS',sortable:false" width="130"><%=rb.getString("JinDu")%></th>
	             	<th data-options="field:'TASK_RESULT',formatter:taskResultFmt,sortable:true" width="130"><%=rb.getString("JieGuo")%></th>
	             	<th data-options="field:'START_TIME',sortable:true" width="170"><%=rb.getString("KaiShiShiJian")%></th>
	                <th data-options="field:'STOP_TIME',sortable:true" width="170"><%=rb.getString("JieShuShiJian")%></th>
	             	<th data-options="field:'USER_CODE',sortable:true" width="200"><%=rb.getString("ChuangJianZhe")%></th>
	            </tr>
            </thead>
        </table>	
	</div>
	<!-- 查看任务结果信息 -->
	<div id="CPElayout_center_progress"  style="border:none;display:none"></div>
		<!-- 新建频点锁 -->
<div class="newPCIContainer">
   	<div class="pciBoxcontainer">
		<div class="omcPageTitleDiv">
			<ul class="omcPageTitleContainer" style="width:80%">
				<li class="default"><%=rb.getString("JiBenXinXi")%></li>
			</ul>
			<ul class="baseInfoBox">
			<li>
				<span><%=rb.getString("RenWuMing")%></span>
				<input type="text" id="cpepcilockTaskName"/>
			</li>
			<li>
				<span><%=rb.getString("ChuangJianZhe")%></span>
				<input type="text" id="cpepcilockCreator" disabled="disabled"/>
			</li>
		</ul>
		</div>
		
	</div>
	<!-- 执行方式  -->
	<div class="pciBoxcontainer executedMode" style="margin-top:10px;">
			<div class="omcPageTitleDiv">
				<ul class="omcPageTitleContainer" style="width:80%">
					<li class="default"><%=rb.getString("ZhiXingFangShi")%></li>
				</ul>
				<div class="execModeBox" padding-left:40px">
					<div class="labelBox">
						<input id="immeDo" type="radio" value="immediately" name="executionWay"/>
						<label for="immeDo"><%=rb.getString("LiJiZhiXing")%></label>
					</div>
	               	<div class="labelBox" >
		               	<input id="scheDo" type="radio" checked="checked" value="schetime" name="executionWay"/>
		               	<label for="scheDo"><%=rb.getString("GuiDingShiJian")%></label>
		               	<div style="margin-top:10px;width:600px;display:inline-block;margin-left:22px;style="height:26px;width:200px;">	    
		               		<input id="CPEPCIscheduleStart" class="easyui-datetimebox border-box border"  data-options="require:true,editable:false" name="scheduleStart" style="height:26px;width:200px;line-height:26px;"/>   
		                </div>
	               	</div>
	            </div>
	            <p id="CPEtimewarning" style="visibility:hidden"><%=rb.getString("ShuRuCuoWu")%></p>
			</div>
	</div>
	<!-- 锁频 -->
	<div class="pciBoxcontainer" style="margin-top:35px;">
		<div class="omcPageTitleDiv">
			<ul class="omcPageTitleContainer" style="width:80%">
				<li class="default"><%=rb.getString("SuoPin")%></li>
			</ul>
			
			<div style="margin-top:30px;margin-left:40px;margin-bottom:20px;">
				<select id="taskScanMode" name="CPE_taskscanMode" class="border border-box item" data-options="editable:false" style="height:30px;width:398px;">
					<option value="fullband">Full Band</option>
					<option value="freqpreferred">Band/Frequency Preferred</option>
					<option value="pcilock">PCI lock</option>
				</select>
			</div>
			<div class="suoPinInputContainer frequencyWay" style="display:none">
				<!-- 选择方式 -->
				<div class="pcilocksignInput selectFreWay" style="">
					<p><%=rb.getString("SuoDingPinDian")%></p>
					<input id="pingDianRangeInput" onblur="verifyInput(this);cancleTrim(this,this.value)"/>
					<span><%=rb.getString("PinDianFanWei")%></span>
				</div>
				<div class="pcilocksignInput selectPCIWay" style="margin-left:50px;">
					<p><%=rb.getString("SuoDingPCI")%></p>
					<input id="PCIRangeInput" onblur="verifyInput(this);cancleTrim(this,this.value)"/>
					<span><%=rb.getString("SpecificPCIFanWei")%></span>
				</div>
			</div>

			<div class="selected_device_box">
				<div class="selected_device_title"><%=rb.getString("YiXuanSheBei")%><%=rb.getString("MaoHao")%></div>
				<div class="selected_device_container" id="selected_device_container">
					
				</div>
				<div class="repeatAlermTitle"><%=rb.getString("YouWeiXiuGaiXiang")%></div>
				<!-- 查询 -->
				<div class="queryGroup" style="margin-left:40px;margin-top:30px;">
					<input id="cpesearchPCI" placeholder="<%=rb.getString("CPEBianMa")%>&nbsp;/&nbsp;IMSI" style="width:50opx"/>
					<b onclick="queryPciLockCellCodeInfo()"></b>
				</div>
				<div style="padding-left:20px;height:380px;width:720px;overflow:auto;margin-left:20px;margin-top:20px;" id="PciLockDiv">
					<%-- 基站列表 --%>
					<table id="CPEPCILockBord" ></table>
				</div>
				<div id="moreDeviceAlarmTitle" style="margin-left:40px;margin-top:10px;color:red;display:none;"><%=rb.getString("ZuiDuoXuanZeSheBei10")%></div>
			</div>
		</div>
	</div>
	<!-- 按钮 -->
	<div class="linkbuttonGroup" style="float:left;margin-left:65px;margin-top:30px;margin-bottom:20px;">
    	<a id="savePCISelectedData" onclick="confirmPCITask()" href="#" class="linkbutton linkbutton_trend"><span><%=rb.getString("QueDing")%></span></a>
    	<a href="#" class="linkbutton linkbutton_nowanna" onclick="resetCPEPCIInput()"><span><%=rb.getString("QuXiao")%></span></a>
   	</div>
   	<div style="display:none" class="successTotask"><%=rb.getString("XinJianRenWuChengGong")%></div>
</div>
	<!-- 查看任务结果信息 -->
	<div id="CPElayout_center_progressess"  style="border:none;display:none"></div>
</div>

<%-- 高级查询 --%>
<div id="toolbar_PCITaskList" class="omcTableTool">
	<div id="faultQuery" class="admin_query_head">
	    <form id="PCIqueryform">
	    	<div class="easyui-query" tips="<%=rb.getString("GaoJiChaXun") %>" 
	    		name="searchText" 
	    		inputId="CPEpciLockSearchText" targetId="CPEPCIQueryDiv" 
	    		placeholder="<%=rb.getString("PCIRenWuMing")%>/<%=rb.getString("ChuangJianZhe")%>" 
	    		data-options="query: queryPciLockTaskList"></div>
	    	<%-- <div class="defaultQuery">
		        <div id="summaryQueryDiv" style="margin-left:0px;">        
			        <input style="margin-left:0px;" id="CPEpciLockSearchText" name="searchText" class="searchInputStyle faultListInput" placeholder="<%=rb.getString("PCIRenWuMing")%>/<%=rb.getString("ChuangJianZhe")%>" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)"/>
			       	<div class="highQueryArrow">	       	
				       	<span class="highQueryTip" id="CPEPCImy_hiddenSpan" onclick="moreQuerySlideFun()" style="cursor:pointer;"><%=rb.getString("GaoJiChaXun") %></span>
				       	<img class="queryHighBtnAlarm query-arrow"  id="CPEPCImoreQueryImg" onclick="moreQuerySlideFun()" flag="1"/>
			       	</div>		
			    </div>
			 <div id="isSuper">
			     <b class="searchResultImgChangeStyle" onclick="queryPciLockTaskList()"></b>
			 </div> 
		    </div> --%>
		    <!-- 隐藏列表 -->
	        <div id="CPEPCIQueryDiv" style="width:100%;padding:20px 20px 10px 40px;display:none;position:absolute;top:68px;left:0px;z-index:100;background:#FFFFFF;-webkit-box-shadow:0px 10px 32px rgba(158,200,222,0.35);">
		        <ul class="inputslist">
		            <li>
		            	<label><%=rb.getString("PCIRenWuMing")%>:</label><br>
		        		<input id="CPEpciTaskName" type="text"  name="CPEpciTaskName" class="border-box border" style="height:26px;width:200px;"/>
		            </li>
		            <li>
		                <label><%=rb.getString("ZhuangTai")%>:</label><br>
		                <select id="CPEpciTaskStatus" class="easyui-combobox border border-box" data-options="editable:false" name="CPEpciTaskStatus" style="height:26px;width:200px;">
		                    <option value=""><%=rb.getString("QuanBu")%></option>
		                    <option value="0"><%=rb.getString("DengDai")%></option>
		                    <option value="1"><%=rb.getString("JinXingZhong")%></option>                    
		                    <option value="2"><%=rb.getString("YiJieShu")%></option>
		                </select>
		            </li>
		            <li>
		            	<label><%=rb.getString("ChuangJianZhe")%>:</label><br>
		            	<input type="text" id="CPEpciUserCode" class="border-box border" name="CPEpciUserCode" style="height:26px;width:200px;"/>
		            </li>
		            <li>
		                <label><%=rb.getString("JieGuo")%>:</label><br>
		                <select id="CPEpciTaskResult" class="easyui-combobox border border-box" data-options="editable:false" name="CPEpciTaskResult" style="height:26px;width:200px;">
		                    <option value=""><%=rb.getString("QuanBu")%></option>
		                    <option value="0"><%=rb.getString("ChengGong")%></option>
		                    <option value="1"><%=rb.getString("BuFenChengGong")%></option>
		                    <option value="2"><%=rb.getString("ShiBai")%></option>
		                </select>
		            </li>
		        </ul>
		        <div class="linkbuttonGroup" style="margin-bottom:20px">
		        	<a href="#" class="linkbutton linkbutton_trend" onclick="queryPCITaskList()"><span><%=rb.getString("ChaXun")%></span></a>
		        	<a href="#" class="linkbutton linkbutton_nowanna" onclick="resetPCIQueryInput()"><span><%=rb.getString("ChaXunChongZhi")%></span></a>
	        	</div>
	        </div>
	    </form>
	</div>	   
</div>

<script type="text/javascript">
    $(function () {
        closeLoading();
        var ele = $("#CPEPCIscheduleStart");
    	disableSelectEarlyTime(ele);  
    	//下拉框该表的时候
    	$("#taskScanMode").change(function() {
    		var scanMode = $("select[name='CPE_taskscanMode']").val();
    		if (scanMode == "pcilock") {
    			$(".frequencyWay").slideDown(300);
    			$(".selectFreWay").slideDown(300);
    			$(".selectPCIWay").slideDown(300);
    		} else if(scanMode == "freqpreferred"){
    			$(".frequencyWay").slideDown(300);
    			$(".selectFreWay").slideDown(300);
    			$(".selectPCIWay").slideUp(300);
    		}else {
    			$(".frequencyWay").slideUp(300);
    			$(".selectFreWay").hide();
    			$(".selectPCIWay").hide();
    		}
    	});
    	
    	$("#CPEpciLockSearchText").bind("keyup", function(e){
    		if (e.keyCode == 13){
    			queryPciLockTaskList();
    		}
    	});
    	
    	$("#cpesearchPCI").bind("keyup", function(e){
    		if (e.keyCode == 13){
    			queryPciLockCellCodeInfo();
    		}
    	});
    	
    	   //点击页面其他位置，隐藏操作下拉选项菜单
        $(document).click(function(e){
            var e = e || window.event;
            var elem = e.target || e.srcElement;
            while(elem){
                if(elem.className == 'operation_more' || elem.className == 'circleBg add_circle' || elem.className == 'showOp' || elem.className == 'slideDiv'){
                    return
                } 
                elem = elem.parentNode;
            }
            $(".showOp").css('display','none');
            $("#choseList").css('display','none');
        	$("#CPElayout_center_progress").hide(400);
        })
      
    	$("#cpepcilockTaskName").val("${addTaskName}");
    	$("#cpepcilockCreator").val("${creator}");
    	
    });
    function pci_lock_cpe_grid_load_success() {
    	$(this).datagrid("enableContextmenuAutoSize");
    }
    function tablePciLockTaskListBeforeLoad(param) {
    	try{
    		var taskName = $("#CPEpciTaskName").val();
    		var taskStatus = $("#CPEpciTaskStatus").combobox("getValue");
    		var userCode = $("#CPEpciUserCode").val();
    		var taskResult = $("#CPEpciTaskResult").combobox("getValue");
    		var searchText = $("#CPEpciLockSearchText").val();
    	} catch(e){
    		return;
    	}
    	if (taskName  && taskName.replace(/\s/g,"").length > 0) {
    		param["taskName"] = taskName;
    	}
    	if (taskStatus  && taskStatus.replace(/\s/g,"").length > 0) {
    		param["taskStatus"] = taskStatus;
    	}
    	if (taskResult  && taskResult.replace(/\s/g,"").length > 0) {
    		param["taskResult"] = taskResult;
    	}
    	if (searchText  && searchText.replace(/\s/g,"").length > 0) {
    		param["searchText"] = searchText;
    	}
    	if (userCode  && userCode.replace(/\s/g,"").length > 0) {
    		param["userCode"] = userCode;
    	}
    	
    }
    
    //新加

    //高级查询下拉列表
/* function moreQuerySlideFun(){
	if($("#CPEPCImoreQueryImg").attr("flag")=="1"){
		$("#CPEPCIQueryDiv").slideDown(500);
		$("#CPEPCImoreQueryImg").attr("flag","0");
		$("#CPEPCImoreQueryImg").addClass('expanded');
	}else{
		$("#CPEPCIQueryDiv").slideUp(400);
		$("#CPEPCImoreQueryImg").attr("flag","1");
		$("#CPEPCImoreQueryImg").removeClass('expanded');
	}	
}
    //高级查询结果
function queryPCITaskList(){
	$("#CPEPCIQueryDiv").slideUp(100);	
	$("#CPEPCImoreQueryImg").removeClass('expanded');
	$("#CPEPCImoreQueryImg").attr("flag","1"); 
	$("#CPEPCImy_hiddenSpan").hide();
	$("#CPEPCITaskList").datagrid('reload');
} */
    
//高级查询重置
function resetPCIQueryInput(){
	$("#CPEpciTaskName").val(null);
	$("#CPEpciUserCode").val(null);
	$("#CPEpciTaskStatus").combobox('setValue', '');
	$("#CPEpciTaskResult").combobox('setValue', '');
}
//高级查询
/* $("#CPEpciLockSearchText").focus(function(){
	$("#CPEPCImy_hiddenSpan").show();
});

$("#CPEPCImoreQueryImg").mouseenter(function(){
	if($("#CPEpciLockSearchText").is(":focus")){
		return;
	}
	$("#CPEPCImy_hiddenSpan").show();
});

$("#CPEPCImoreQueryImg").mouseleave(function(){
	if($("#CPEpciLockSearchText").is(":focus")){
		return;
	}
	if($("#CPEPCIQueryDiv").is(":visible")){
		return;
	}
	$("#CPEPCImy_hiddenSpan").hide();
});

$("#CPEPCImy_hiddenSpan").mouseleave(function(){
	if($("#CPEpciLockSearchText").is(":focus")){
		return;
	}
	if($("#CPEPCIQueryDiv").is(":visible")){
		return;
	}
	$("#CPEPCImy_hiddenSpan").hide();
}); */
//查询
function queryPciLockTaskList(){
	$("#CPEPCITaskList").datagrid('reload',{timeZone:timeZone});
}
    //设置操作列单元格样式 
function setStyle(){
	return 'position:relative';
}
    /**
 * 格式化操作
 */
function pciLockTaskFormatter(value, rowData, rowIndex){
	var task_id = rowData.TASK_ID;
    var task_type = "pci";
	var task_progress = rowData.TASK_PROGRESS;
	
	var JieGuo = '<%=rb.getString("JieGuo")%>';
	var JiHuo = '<%=rb.getString("JiHuo")%>';
	var GuaQi = '<%=rb.getString("GuaQi")%>';
	var ZhongZhi = '<%=rb.getString("ZhongZhi")%>';
	var ShanChu = '<%=rb.getString("ShanChu")%>';
	value = "<div class='operation_more' title='"+ CaoZuo+"' onclick='choseOp("+ task_id +",this)'></div>";
	
	var opt="";
	opt = opt + "<div class='titleDiv operation_result' title='"+JieGuo+"' onclick='showPciLockTaskDetail()' style='margin-left:15px;'>"+JieGuo+"</div>";
	if(task_progress != 2){//当前任务没有结束，终止图标可用
		opt = opt + "<div class='titleDiv titleIcon_terminate_click CODE_CPE_PCI_LOCK hidden' title='"+ZhongZhi+"' style='margin-left:15px;' onclick='terminateTask(\"" + task_id + "\",\"" + task_type +"\")'>"+ZhongZhi+"</div>";
	}else{
		opt = opt + "<div class='titleDiv titleIcon_terminate_disabled CODE_CPE_PCI_LOCK hidden' title='"+ZhongZhi+"' style='margin-left:15px;'>"+ZhongZhi+"</div>";
	}
	
	if(task_progress != 1){//当前任务不在进行中，删除图标可用
		opt = opt + "<div class='titleDiv titleIcon_delete_click CODE_CPE_PCI_LOCK hidden' title='"+ShanChu+"' style='margin-left:15px;' onclick='delTask(\"" + task_id + "\",\"" + task_type +"\")'>"+ShanChu+"</div>";
	}else{  
		opt = opt + "<div class='titleDiv titleIcon_delete_disabled CODE_CPE_PCI_LOCK hidden' title='"+ShanChu+"' style='margin-left:15px;'>"+ShanChu+"</div>";
	}
	value = value + "<div class='showOp'>"+ opt +"</div>";
	return value;
}
//锁频格式化
function frequencyFormatter(value, rowData, rowIndex){
	if(value == "pcilock"){
		return "<span>"+'<%=rb.getString("SuoPCI")%>'+"</span>"
	}else if(value == "fullband"){
		return "<span>"+'<%=rb.getString("BuSuoPin")%>'+"</span>"
	}else{
		return "<span>"+'<%=rb.getString("SuoPin")%>'+"</span>"
	}
	
}
//点击行内【更多】按钮，下拉显示操作选项 
function choseOp(idVal,e){
	var thisTop = $(e).offset().top;
	var allHeight = $(document).height();
	var indexRow = $("#CPEPCITaskList").datagrid("getRowIndex",idVal);
	var rowHeight = $("#PCItaskTable .datagrid-view2").find("tr[datagrid-row-index="+indexRow+"]").height();
	
	if((allHeight - thisTop) < 240){
		$(e).next(".showOp").css("bottom",rowHeight+"px");
	}else{
		$(e).next().css("top",rowHeight+"px");
	}
	
	$("#choseList").css('display','none');
	$(".showOp").hide();
 	$("#CPElayout_center_progress").hide(400);
	$(e).next().fadeToggle(300);
}
//任务状态格式化
function taskProgressFmt(value, rowData, rowIndex) {
	if (value == "0") {
		return "<%=rb.getString("DengDai")%>";
	} else if (value == "1") {
		return "<%=rb.getString("JinXingZhong")%>";
	} else if (value == "2") {
		return "<%=rb.getString("YiJieShu")%>";
	}
}
//任务执行结果格式化
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
//显示任务进度
function showPciLockTaskDetail() {
	$("#CPEPCITaskList").datagrid("reload");
	var selectedTask = $("#CPEPCITaskList").datagrid("getSelected");
	if (!selectedTask) {
		return;
	}
	var task_id = selectedTask["TASK_ID"];
	var task_type = selectedTask["TYPE"];
	
	if($("#CPElayout_center_progress").css('display') == 'block'){
	}else{
		$("#CPElayout_center_progress").show(400).fadeIn(400);
	}
	
	$(".showOp").slideUp(100);
    $("#CPElayout_center_progress").panel({
        href: "${ctx}/cpe/strategy/toPciLockTaskProgress.action?task_id=" + task_id + "&type=" + task_type
    });
}
//终止任务
function terminateTask(idVal,type) {
	/* RenWuYiJieShu */
	var params = {};
	params["taskId"] = idVal;
	params["type"] = type;
	$.post("${ctx}/cpe/strategy/terminatePciLockTask.action", params, function(data) {
		if (data["success"]) {
			$("#CPEPCITaskList").datagrid("reload");
		}else{
			$.messager.alert(TiShi, data["message"]);
		}
	}, "json");
}

//删除任务
function delTask(idVal,type){
	var params = {};
	params["taskId"] = idVal;
	params["type"] = type;
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChuRenWu")%>", function(r) {
     if (r) {
         $.post("${ctx}/cpe/strategy/delPciLockTask.action", params, function(data) {
             if (data["success"]) {
                 $("#CPEPCITaskList").datagrid("reload");
             } else {
                 $.messager.alert(TiShi, data["message"]);
             }
         }, "json");
     }
 }).addClass("seriousConfirm");
}


var showpcilockFlag = true
function newcpePCILockTask(){
	if(showpcilockFlag){
		$(".newPCIContainer").slideDown(300);
		$(".titleButtonText").html("<%=rb.getString("GuanBi")%>");
		$(".circleBg").removeClass("add_circle");
		$(".circleBg").addClass("close_circle");
		$("#cpepciTitle").html("<%=rb.getString("XinJianPinDianSuoRenWu")%>");
		showpcilockFlag = false;
		$("#CPEPCILockBord").datagrid({
			url: '${ctx}/cell/CPE/queryCpeInfosList.action?type=0',
			queryParams:{like_fields:"SERIAL_NUMBER,IMSI"},
			singleSelect:false,
			fit:true,
			fitColumns:true,
			border:true,
			rownumbers:true,
			pagePosition:'bottom',
			pageSize : 100,
			pageList : [100],
			idField:'CPE_CODE',
			toolbar:'#toolbar_GridCell_cellParam',
			/* onLoadSuccess:gridCellParamDatagridLoadSuccess,
			onLoadError:datagridLoadError, */
			onBeforeLoad:beforeLocad_CellCodeList,
			onCheck:selectCPEcheck,
			onUncheck:removeCPEcheck,
			onCheckAll: selectedAllCPEPCI,
			onUncheckAll:emptySelectedCPE,
			//onBeforeCheck:disableSelect,
			//onBeforeSelect:disableSelect,
			pagination : true,
			striped: true,
			
			columns: [[
				{field: 'ck',checkbox:true},
				{field: 'CPE_CODE', hidden: true},
				{field: 'CONNECTION_STATUS',fixed:true,width: 30,formatter:connStatusFormatter}, 
				{field: 'SERIAL_NUMBER',sortable:true,width: 100, title: '<%=rb.getString("CPEBianMa")%>'},
				{field: 'IMSI',sortable:true,width: 100, title: 'IMSI'}
			]],
			
		});
		$("#CPEPCIscheduleStart").datebox({
			onChange:function(){
				$("#CPEPCIscheduleStart").next().css({"border-color":"#85A8BF"});
				$("#CPEtimewarning").css("visibility","hidden");	
			}
		})
	}else{
		resetCPEPCIInput();
	}
	
}
//下拉列表的表格函数
function beforeLocad_CellCodeList(param){
	var search_text = $("#cpesearchPCI").val();
	if(search_text != ""){
		param["search_text"] = search_text;
	}
}
//全选cpepcilock
function selectedAllCPEPCI(rows){
	$("#selected_device_container").empty();
	var cpeSerialNumber = "";
	var cpeCode = "";
	var cpeRowNumber;
	for(var i = 0;i<rows.length;i++){
		cpeCode = rows[i].CPE_CODE;
		cpeSerialNumber = rows[i].SERIAL_NUMBER;
		cpeRowNumber = i;
		var appendString = '<div class="selectedDetail_mana" rowNumber="'+cpeRowNumber+'" cpecode="'+cpeCode+'"  cpeserialnumber="'+cpeSerialNumber+'"><div style="display:inline-block;margin-right:15px;margin-left:5px;">' + cpeSerialNumber 
		+ '</div><div class="operationDiv titleIcon_selection_del" onclick="delete_selected_CPE(this)"></div></div>';
		$("#selected_device_container").append(appendString);
	}
}
//取消全选
function emptySelectedCPE(rows){
	$("#selected_device_container").empty();
}
//勾选行
function selectCPEcheck(index,row){
	var cpeSerialNumber = row.SERIAL_NUMBER;
	var cpeRowNumber = index;
	var cpeCode = row.CPE_CODE;
	var appendString = '<div class="selectedDetail_mana" rowNumber="'+cpeRowNumber+'" cpecode="'+cpeCode+'"  cpeserialnumber="'+cpeSerialNumber+'"><div style="display:inline-block;margin-right:15px;margin-left:5px;">' + cpeSerialNumber 
		+ '</div><div class="operationDiv titleIcon_selection_del" onclick="delete_selected_CPE(this)"></div></div>';
	$("#selected_device_container").append(appendString);
		
}
//取消行
function removeCPEcheck(index,row){
	var cpeCode = row.CPE_CODE;
	var allSelectCPE = $("#selected_device_container").children();
	for(var i=0;i<allSelectCPE.length;i++){
		if($(allSelectCPE[i]).attr("cpecode") == cpeCode){
			$(allSelectCPE[i]).remove();
			break;
		}
	}
}
//X号清除选择的行
function delete_selected_CPE(ele){
	var unCheckRowNumber = $(ele).parent().attr("rowNumber");
	$(ele).parent().remove();
	$("#CPEPCILockBord").datagrid("uncheckRow",unCheckRowNumber);
}
function queryPciLockCellCodeInfo(){
	$("#CPEPCILockBord").datagrid('reload');
}
//下拉列表频点范围凭此范围校验
function verifyInput(ele){
	var reg = /\s/;
	var inputvla = $(ele).val().trim();
	if($(ele).attr("id") == "pingDianRangeInput"){
		if(inputvla<0 || inputvla>65535){
			$(ele).css("borderColor","#CC0000")
			$(ele).next().css("display","inline-block");
			$(ele).next().text("<%=rb.getString("PinDianFanWei")%>");
		}else if(reg.test(inputvla)){
			$(ele).css("borderColor","#CC0000")
			$(ele).next().css("display","inline-block");
			$(ele).next().text("<%=rb.getString("PinDianFanWei")%>");
		}else if((inputvla>=0&&inputvla<=65535&&inputvla!="")){
			$(ele).css("borderColor","#85A8BF")
			$(ele).next().css("display","none");
		}else if(isNaN(inputvla)){
			$(ele).css("borderColor","#CC0000")
			$(ele).next().css("display","inline-block");
			$(ele).next().text("<%=rb.getString("PinDianFanWei")%>");
		}else if(inputvla == "" ){
			$(ele).css("borderColor","#CC0000")
			$(ele).next().css("display","inline-block");
			$(ele).next().text("<%=rb.getString("PinDianFanWei")%>");
		}
	}else{
		if(inputvla<0 || inputvla>503){
			$(ele).css("borderColor","#CC0000")
			$(ele).next().css("display","inline-block");
			$(ele).next().text("<%=rb.getString("SpecificPCIFanWei")%>");
		}else if(reg.test(inputvla)){
			$(ele).css("borderColor","#CC0000")
			$(ele).next().css("display","inline-block");
			$(ele).next().text("<%=rb.getString("SpecificPCIFanWei")%>");
		}else if((inputvla>=0&&inputvla<=503&&inputvla!="")){
			$(ele).css("borderColor","#85A8BF")
			$(ele).next().css("display","none");
		}else if(isNaN(inputvla)){
			$(ele).css("borderColor","#CC0000")
			$(ele).next().css("display","inline-block");
			$(ele).next().text("<%=rb.getString("SpecificPCIFanWei")%>");
		}else if(inputvla == "" ){
			$(ele).css("borderColor","#CC0000")
			$(ele).next().css("display","inline-block");
			$(ele).next().text("<%=rb.getString("SpecificPCIFanWei")%>");
		}
	}
}
//确定信息提交
function confirmPCITask(){
	var params = {};
	params.timeZone = timeZone;
	var taskName = $("#cpepcilockTaskName").val();
	var creator = $("#cpepcilockCreator").val();
	params.taskName = taskName;
	params.creator = creator;
	var selectedItemLength = $(".selected_device_container .selectListItem").length;
	var strStartTime = $("#CPEPCIscheduleStart").combobox('getValue');
	var selectVal = $("input:radio[name='executionWay']:checked").val();
	if(selectVal == "schetime"){
		params.status = "timing";
		if(strStartTime == ""){
			$(".newPCIContainer").animate({
				scrollTop:270
			});
			$("#CPEPCIscheduleStart").next().css({"border-color":"red"});
			$("#CPEtimewarning").css("visibility","visible");	
			return;
		} else{
			$("#CPEPCIscheduleStart").next().css({"border-color":"#85A8BF"});
			$("#CPEtimewarning").css("visibility","hidden");	
			params.time = strStartTime
		}
	}else if(selectVal == "immediately"){
		params.status = "active";
		$("#CPEtimewarning").css("visibility","hidden");
		$("#CPEPCIscheduleStart").next().css({"border-color":"#85A8BF"});	
		//传递的参数
	}else if(typeof selectVal == "undefined"){
		$(".newPCIContainer").animate({
			scrollTop:290
		}) 
		return;
	} 
	var reg = /\s/;;
	var frequency = $("#pingDianRangeInput").val().trim();
	var pci_val = $("#PCIRangeInput").val().trim();
	//根据下拉框节点判断
	if($("select[name='CPE_taskscanMode']").val() == "pcilock"){
		params.lock_mode = "pcilock";
		if(frequency=="" || reg.test(frequency)){
			$("#pingDianRangeInput").css("border-color","#CC0000");
			$("#pingDianRangeInput").siblings("span").css("display","inline-block");
			$("#pingDianRangeInput").siblings("span").text("<%=rb.getString("PinDianFanWei")%>");
			return;
		}
		if(pci_val=="" || reg.test(pci_val)){
			$("#PCIRangeInput").css("border-color","#CC0000");
			$("#PCIRangeInput").siblings("span").css("display","inline-block");
			$("#PCIRangeInput").siblings("span").text("<%=rb.getString("SpecificPCIFanWei")%>");
			return;
		}
		if(frequency!=""&&pci_val!=""){
			$("#pingDianRangeInput").siblings("span").css("display","none");
			$("#PCIRangeInput").siblings("span").css("display","none");
			params.frequency = $("#pingDianRangeInput").val();
			params.pci_val = $("#PCIRangeInput").val();
		}else{
			return;
		}
	}else if($("select[name='CPE_taskscanMode']").val() == "freqpreferred"){
		params.lock_mode = "freqpreferred";
		if(frequency=="" || reg.test(frequency)){
			$("#pingDianRangeInput").css("border-color","#CC0000");
			$("#pingDianRangeInput").siblings("span").css("display","inline-block");
			$("#pingDianRangeInput").siblings("span").text("<%=rb.getString("PinDianFanWei")%>");
			return;
		}
		if(frequency!=""){
			$("#pingDianRangeInput").siblings("span").css("display","none");
			$("#PCIRangeInput").siblings("span").css("display","none");
			params.frequency = $("#pingDianRangeInput").val();
		}else{
			return;
		}
	}else{
		params.lock_mode = "fullband";
	}
	

   var cpesString="";
   var selectCpeDevice = $("#selected_device_container").children();
   if(selectCpeDevice.length<=0){
	   $.messager.alert(TiShi, "<%=rb.getString("QingXuanZeSheBei")%>");
   	   return;
   }
   for(var i=0;i<selectCpeDevice.length;i++){
	   cpesString += ($(selectCpeDevice[i]).attr("cpecode")+",");
   }
   cpesString=cpesString.substring(0,cpesString.length-1),
   params.cpe_codes = cpesString;
   $.post("${ctx}/cpe/strategy/addTask.action", params, function(data){
       if (data["success"]) {
           $(".successTotask").css("display","block")
           setTimeout(function(){
        	   $(".newPCIContainer").slideUp(300);
    	   	   $(".titleButtonText").html("<%=rb.getString("TianJia")%>");
    	   	   $(".circleBg").addClass("add_circle");
    	   	   $(".circleBg").removeClass("close_circle");
    	   	   $("#cpepciTitle").html("<%=rb.getString("PinDianSuo")%>");
    	   	   $(".successTotask").css("display","none")
    	   	   showpcilockFlag = true;
    	   	   resetCPEPCIInput();
           },1500)
          
       } else {
           $.messager.alert(TiShi, data["message"]);
       }
       $("#CPEPCITaskList").datagrid("reload"); 
   }, "json");
   
   
}
function resetCPEPCIInput(){
	$(".newPCIContainer").slideUp(300);
	$(".titleButtonText").html("<%=rb.getString("TianJia")%>");
	$(".circleBg").addClass("add_circle");
	$(".circleBg").removeClass("close_circle");
	$("#cpepciTitle").html("<%=rb.getString("PinDianSuo")%>");
	$("#pingDianRangeInput").siblings("span").hide();
	$("#PCIRangeInput").siblings("span").hide();
	$("#selected_device_container").empty();
	$("#PCIRangeInput").css("border-color","#85A8BF");
	$("#pingDianRangeInput").css("border-color","#85A8BF");
	$("#pingDianRangeInput").val("");
	$("#PCIRangeInput").val("");
	$("#CPEPCIscheduleStart").combobox("clear");
	$("#CPEPCIscheduleStart").next().css({"border-color":"#85A8BF"});
	$("#CPEtimewarning").css("visibility","hidden");
	$(".successTotask").css("display","none");
	$("#CPEPCILockBord").datagrid("uncheckAll");
	showpcilockFlag = true;
}
//输入框自动去前后空格
function cancleTrim(dom,str){
	$a = str.replace(/(^\s*)|(\s*$)/g,"");
	$(dom).val($a);
}
</script>