<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<style type="text/css">
.panelDefault .logsTabsMainPage{
	/* top:40px;
	left:20px; */
}
.logsTabsMainPage .datagrid-toolbar{
	padding:0 !important;
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
	padding:20px 20px 10px 20px;
	/* position:absolute;  */
	top:0px;
	box-shadow:0 2px 6px 0 rgba(171,191,221,0) !important;
}
.defaultQuery{
	padding:20px 0 20px 20px;
}
.infoDetailStyle{
	top:48px;
	display:none;
	position:absolute;
	left:0px;
	z-index:100;
	width:100%;
	background:#FFFFFF;
	padding-bottom:10px;
	padding-top:20px;
	-webkit-box-shadow:0px 10px 32px rgba(158,200,222,0.35);
}
.inputslist label {
    margin: 0px 8px 0px 0px;
}
.inputslist li{
	float: left;
    height: 35px;
    margin-right:80px;
    margin-bottom:30px;
    margin-left:20px;
    margin-top:10px;

}
.textbox.combo .textbox-text {
	padding: 0 4px !important;
}
.highQueryTip{
	cursor:pointer;
}
</style>

<%--系统-日志   系统操作日志记录页面 --%>
<div class="panelDefault">
	<!-- 右上角导出按钮 -->
	<div class="circleIcon" style="right:45px;">		
		<span class="el-icon el-icon-circle-export" onclick="exportLogsTurn()"></span>
		<div class="titleButtonText"><%=rb.getString("DaoChu")%></div>
	</div>
	<div class="tabsTitle omcLogLists" id="omcLogTabsDiv" style='color:#333'>
		<span tabtit="opLogDiv" class="active CODE_SYSTEM_LOGS_OPERATION hidden visible" logtype="op"><%=rb.getString("CaoZuoRiZhi")%></span>
		<%-- <c:if test="${isElfCell == 1}"> 
			<span tabtit="enbUpgradeLogDiv" logtype="up"><%=rb.getString("JiZhanShengJiJiLu")%></span>
		</c:if>  --%>
		<span tabtit="securityLogDiv" class="CODE_SYSTEM_LOGS_SECURITY hidden visible" logtype="sec"><%=rb.getString("AnQuanRiZhi")%></span>
		<span tabtit="sysLogDiv" class="systemLog hidden visible" logtype="sys"><%=rb.getString("XiTongRiZhi")%></span>
	</div>
	<div class="tabsContentDiv logsTabsMainPage">
		<div class="opLogDiv" style="display:block;">
			<table id="tabelOperationNote"></table>
		</div>
		
		<div class="securityLogDiv">
			<table id="omcSecurityLogGrid"></table>
		</div>
		<div class="sysLogDiv">
            <table id="omcSystemLogGrid" class="easyui-datagrid"
                data-options="border:false,fit:true,fitColumns:true,singleSelect:true,striped: true,rownumbers:true,
                    url: '${ctx}/system/logmange/system/getSystemLogPageList.action',queryParams : {timeZone : timeZone},
                    toolbar: '#toolbar_omcSystemLogGrid',pagination:true,onLoadSuccess:gridOnLoadSuccessForAutoSize">
                <thead>
                    <tr>
                        <th data-options="field:'serial_number',sortable:true" width="100"><%=rb.getString("Title_SheBeiBianMa")%></th>
                        <th data-options="field:'name',sortable:true" width="100"><%=rb.getString("Title_SheBeiMingCheng")%></th>
                        <th data-options="field:'device_type',sortable:true" width="100"><%=rb.getString("Title_SheBeiLeixing")%></th>
                        <th data-options="field:'operate_ip',sortable:true" width="100"><%=rb.getString("CaoZuoIPAddress")%></th>
                        <th data-options="field:'operation_name',sortable:true" width="150"><%=rb.getString("CaoZuoLeiXing")%></th>
                        <th data-options="field:'detail'" width="200"><%=rb.getString("JinDu")%></th>
                        <th data-options="field:'op_start_time',sortable:true" width="100"><%=rb.getString("ShiJian")%></th>
                    </tr>
                </thead>
            </table>
        </div>
	</div>
</div>

<!-- 工具栏-操作日志 -->
<div id="toolbar_operationNote" class="admin_query_head">
    <form id="form_omcOperationLog" class="higeQuery_form"> 
    	<div class="easyui-query" tips="<%=rb.getString("GaoJiChaXun") %>" 
    		name="search_text" 
    		inputId="operateName_1" targetId="operationNoteDiv" 
    		placeholder="<%=rb.getString("CaoZuoMingCheng")%>" 
    		data-options="query: vagueQueryOperList"></div>
    	<%-- <div class="defaultQuery">
            <div>
            	<input class="operateName" id="operateName_1" name="search_text" class="border border-box" style="height:22px;width:300px;padding-left:10px;" placeholder="<%=rb.getString("CaoZuoMingCheng")%>">
	            <input id="operateName_1" name="search_text" style="margin-left:0px;" placeholder="<%=rb.getString("CaoZuoMingCheng")%>" class="searchInputStyle omcLogInputSearch" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)">
	            <div class="highQueryArrow">
	            	<span class="highQueryTip"  id="highQueryTip_1" onclick="moreQueryImgFun1()"><%=rb.getString("GaoJiChaXun")%></span>
	            	<img class="queryHighBtnLog query-arrow" id="queryHighBtn_1" onclick="moreQueryImgFun1()" style="margin-left:2px;" flag="1">
	            </div>
            </div>
            <div id="isSuper">
             	<b class="searchResultImgChangeStyle" onclick="vagueQueryOperList()"></b>
            </div>
    	</div> --%>
    	<div id="operationNoteDiv" class="infoDetailStyle" >    	
	        <ul class="inputslist" >
	            <li>
	                <label for="sel_op_name_op" style="display:inline-block;margin-bottom:5px;"><%=rb.getString("CaoZuoMingCheng")%>:</label><br>
	                <input id="sel_op_name_op" name="op_id" class="easyui-combobox border border-box sel_op_name_1" data-options="editable:false" style="height: 26px;width:200px;">
	            </li>     
                <li class="operatorItem">
                    <label for="sel_operator_name_op" style="display:inline-block;margin-bottom:5px;"><%=rb.getString("YunYingShang")%>:</label><br>
                    <input id="sel_operator_name_op" name="operator_code" class="easyui-combobox border border-box sel_ope_name_1" data-options="editable:false" style="height: 26px;width:200px;">
                </li>
	            <li>
	                <label for="sel_user_code_op" style="display:inline-block;margin-bottom:5px;"><%=rb.getString("YongHuMingCheng")%>:</label><br>
	                <input id="sel_user_code_op" name="user_code" class="easyui-combobox border border-box sel_user_code_1" style="height: 26px;width:200px;">
	            </li>
	            <li>
	            	<label for="sel_operate_ip" style="display:inline-block;margin-bottom:5px;"><%=rb.getString("CaoZuoIPAddress")%>:</label><br>
	                <input id="sel_operate_ip" name="operate_ip" class="border border-box sel_operate_ip_1" style="height: 26px;width:200px;">
	            </li>
	            <li><label for="sel_detail" style="display:inline-block;margin-bottom:5px;"><%=rb.getString("XiangXiJiLu")%>:</label><br>
	                <input id="sel_detail" name="detail" class="border border-box sel_detail_1" style="height: 26px;width:200px;">
	            </li>
	            <li>
	            	<label for="sel_op_start_time" style="display:inline-block;margin-bottom:5px;"><%=rb.getString("KaiShiShiJian")%>:</label><br>
	                <input id="sel_op_start_time" name="op_start_time" class="easyui-datetimebox border border-box sel_op_start_time_1" data-options="editable:false" style="height: 26px;width:200px;">
	            </li>
	            <li>
	            	<label for="sel_op_end_time" style="display:inline-block;margin-bottom:5px;"><%=rb.getString("JieShuShiJian")%>:</label><br>
	                <input id="sel_op_end_time" name="op_end_time" class="easyui-datetimebox border border-box  sel_op_end_time_1" data-options="editable:false"  style="height: 26px;width:200px;">
	            </li>
	        </ul>
	        <div class="windowButtonGroup" style="margin-left:18px;margin-bottom:10px;float:left"> 
	        	<a href="#" class="linkbutton linkbutton_trend" onclick="accurateQueryOperList()" style="float:left;"><span><%=rb.getString("ChaXun")%></span></a>
	        	<a href="#" class="linkbutton linkbutton_nowanna" style="float:left;" onclick="resetoperationNoteQueryInput()"><span style="min-width:auto;padding:0 20px"><%=rb.getString("ChaXunChongZhi")%></span></a>
	        </div>
	        
    	</div>
    </form>
</div>

<!-- 工具栏-基站升级和自配置日志 -->
<%-- <c:if test="${isElfCell == 1}">
	<div id="toolbar_tableCellUpgradeNote" class="admin_query_head">
		<form id="form_tableCellUpgradeNote"  class="higeQuery_form">
			<div class="defaultQuery">
	            <div>
		            <input id="sel_sn" name="sn" style="margin-left:0px;" placeholder="<%=rb.getString("XiaoZhanBianMa")%>" class="searchInputStyle omcLogInputSearch" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)">
		            <div class="highQueryArrow">
		            	<span class="highQueryTip" id="highQueryTip_2" onclick="moreQueryImgFun2()"><%=rb.getString("GaoJiChaXun")%></span>
		            	<img class="queryHighBtnLog" id="queryHighBtn_2" onclick="moreQueryImgFun2()" flag="1">
		            </div>
	            </div>
	            <div>
	            	<b class="searchResultImgChangeStyle" onclick="vagueQueryUpgradeHistoryList()"></b>
	            </div>
	    	</div>
	    	<div id="tableCellUpgradeNoteDiv"  class="infoDetailStyle">	    	
				<ul class="inputslist" >
					<li>   
						<label for="sel_ip_addr" style="display:inline-block;margin-bottom:5px;"><%=rb.getString("IPDiZhi")%>:</label><br>
						<input id="sel_ip_addr" name="ip_addr" class="border border-box sel_ip_addr_2" style="height: 26px;width:200px;">
					</li>
					<li>
						<label for="sel_mode_name" style="display:inline-block;margin-bottom:5px;"><%=rb.getString("MoKuaiMingCheng")%>:</label><br>
						<select class="easyui-combobox border-box border sel_mode_name_2" name="mode_name" id="sel_mode_name" style="height:26px;width:200px;">
							<option value="0" selected="selected"><%=rb.getString("QuanBu")%></option>
							<option value="1"><%=rb.getString("ShengJiCeLue")%></option>
							<option value="2"><%=rb.getString("SoftwareZiDongShengJi")%></option>
							<option value="3"><%=rb.getString("ParamZiDongPeiZhi")%></option>
						</select>
					</li>
					<li>
						<label for="sel_ori_version" style="display:inline-block;margin-bottom:5px;"><%=rb.getString("ChuShiBanBen")%>:</label><br>
						<input id="sel_ori_version" name="ori_version" class="border border-box sel_ori_version_2" style="height:26px;width:200px;">
					</li>
					<li>
						<label for="sel_dest_version" style="display:inline-block;margin-bottom:5px;"><%=rb.getString("ShengJiBanBen")%>:</label><br>
						<input id="sel_dest_version" name="dest_version" class="border border-box sel_dest_version_2" style="height:26px;width:200px;">
					</li>
					<li style="width:500px">
						<label for="sel_op_start_time"><%=rb.getString("GengXinShiJian")%>:</label><br>
						<div style="margin-top:14px;width:600px">
							<input id="sel_op_start_time" name="upgrade_op_start_time" class="easyui-datetimebox border border-box sel_op_start_time_2" data-options="editable:false" 
								style="height:26px;width:200px;"/>
							<span style="margin:0 40px;">--</span>
							<input id="sel_op_end_time" name="upgrade_op_end_time" class="easyui-datetimebox border border-box sel_op_end_time_2" data-options="editable:false" 
								style="height:26px;width:200px;"/>
						</div>
					</li>		
				</ul>
				<div class="windowButtonGroup" style="margin-left:18px;margin-bottom:10px;float:left">
					<a href="#" class="linkbutton linkbutton_trend" onclick="accurateQueryUpgradeHistoryList()" style="float:left;"><span><%=rb.getString("ChaXun")%></span></a>
					<a href="#" class="linkbutton linkbutton_nowanna" style="float:left;" onclick="resetUpgradeNoteQueryInput()"><span style="min-width:auto;padding:0 20px"><%=rb.getString("ChaXunChongZhi")%></span></a>
				</div>
				
	    	</div>
		</form>
	</div>
</c:if> --%>

<%-- 导出基站升级配置历史记录 --%>
<form id="formExportCellUpgradeLogData" style="display:none" method="post" action="">
	<%--已选择的日志查询参数 --%>
	<input type="hidden" value="" name="upgrade_form_timeZone" id="upgrade_form_timeZone"/>
	<input type="hidden" value="" name="upgrade_form_sn" id="upgrade_form_sn"/>
	<input type="hidden" value="" name="upgrade_form_ip_addr" id="upgrade_form_ip_addr" />
	<input type="hidden" value="" name="upgrade_form_task_name" id="upgrade_form_task_name"/>
	<input type="hidden" value="" name="upgrade_form_mode_name" id="upgrade_form_mode_name"/>
	<input type="hidden" value="" name="upgrade_form_ori_version" id="upgrade_form_ori_version"/>
	<input type="hidden" value="" name="upgrade_form_dest_version" id="upgrade_form_dest_version"/>
	<input type="hidden" value="" name="upgrade_form_op_start_time" id="upgrade_form_op_start_time"/>
	<input type="hidden" value="" name="upgrade_form_op_end_time" id="upgrade_form_op_end_time"/>
	<input type="hidden" value="" name="upgrade_form_result" id="upgrade_form_result"/>
</form>

<!-- 安全日志记录查询面板-->
<div id="toolbar_omcSecurityLogGrid" class="admin_query_head">
    <form id="form_omcSecurityLog" class="higeQuery_form"> 
    	<div class="easyui-query" tips="<%=rb.getString("GaoJiChaXun") %>" 
    		name="search_text" 
    		inputId="operateName_3" targetId="omcSecurityLogGridDiv" 
    		placeholder="<%=rb.getString("CaoZuoMingCheng")%>" 
    		data-options="query: vagueQuerySecurityLogList"></div>
        <%-- <div class="defaultQuery">
            <div>
                <input id="operateName_3" name="search_text" style="margin-left:0px;" placeholder="<%=rb.getString("CaoZuoMingCheng")%>" class="searchInputStyle omcLogInputSearch" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)">
                <div class="highQueryArrow">
                    <span class="highQueryTip" id="highQueryTip_3" onclick="moreQueryImgFun3()"><%=rb.getString("GaoJiChaXun")%></span>
                    <img class="queryHighBtnLog query-arrow" id="queryHighBtn_3" onclick="moreQueryImgFun3()" flag="1">
                </div>
            </div>
            <div id="isSuper">
                <b class="searchResultImgChangeStyle" onclick="vagueQuerySecurityLogList()"></b>
            </div>
        </div> --%>
        <div id="omcSecurityLogGridDiv" class="infoDetailStyle">        
	        <ul class="inputslist" >
	            <li>
	                <label for="sel_op_name_sec" style="display:inline-block;margin-bottom:5px;"><%=rb.getString("CaoZuoMingCheng")%>:</label><br>
	                <input id="sel_op_name_sec" name="op_id" class="border border-box sel_op_name_3" style="height: 26px;width:200px;">
	            </li>
                <li class="operatorItem">
                    <label for="sel_operator_name_sec" style="display:inline-block;margin-bottom:5px;"><%=rb.getString("YunYingShang")%>:</label><br>
                    <input id="sel_operator_name_sec" name="operator_code" class="border border-box sel_ope_name_3" style="height: 26px;width:200px;">
                </li>
	            <li>
	                <label style="display:inline-block;margin-bottom:5px;"><%=rb.getString("YongHuMingCheng")%>:</label><br>
	                <input id="sel_user_code_sec" name="user_code" class="easyui-combobox border border-box user_code_3" style="height: 26px;width:200px;">
	            </li>
	            <li>
	                <label style="display:inline-block;margin-bottom:5px;"><%=rb.getString("CaoZuoIPAddress")%>:</label><br>
	                <input name="operate_ip" class="border border-box operate_ip_3" style="height: 26px;width:200px;"></li>
	            <li>
	                <label style="display:inline-block;margin-bottom:5px;"><%=rb.getString("XiangXiJiLu")%>:</label><br>
	                <input name="detail" class="border border-box detail_3" style="height: 26px;width:200px;">
	            </li>
	            <li>
	                <label style="display:inline-block;margin-bottom:5px;"><%=rb.getString("KaiShiShiJian")%>:</label><br>
	                <input name="op_start_time" class="easyui-datetimebox border border-box op_start_time_3" data-options="editable:false"  style="height: 26px;width:200px;">
	            </li>
	            <li>
	                <label style="display:inline-block;margin-bottom:5px;"><%=rb.getString("JieShuShiJian")%>:</label><br>
	                <input name="op_end_time" class="easyui-datetimebox border border-box op_end_time_3" data-options="editable:false" style="height: 26px;width:200px;">
	            </li>
	        </ul>
	        <div class="windowButtonGroup" style="margin-left:18px;margin-bottom:10px;float:left">
	        	<a href="#" class="linkbutton linkbutton_trend" onclick="accurateQuerySecurityLogList()" style="float:left;"><span><%=rb.getString("ChaXun")%></span></a> 
	        	<a href="#" class="linkbutton linkbutton_nowanna" style="float:left;" onclick="resetSecurityLogQueryInput()"><span style="min-width:auto;padding:0 20px"><%=rb.getString("ChaXunChongZhi")%></span></a>
	        </div>
	        
        </div>
    </form>
</div>

<!-- 系统日志记录查询面板-->
<div id="toolbar_omcSystemLogGrid" class="admin_query_head">
    <form id="form_omcSystemLog" class="higeQuery_form"> 
    	<div class="easyui-query" tips="<%=rb.getString("GaoJiChaXun") %>" 
    		name="search_text" 
    		inputId="operateName_4" targetId="omcSystemLogDiv" 
    		placeholder="<%=rb.getString("Title_SheBeiBianMa")%>" 
    		data-options="query: vagueQuerySystemLogList"></div>
        <%-- <div class="defaultQuery">
            <div>
                <input id="operateName_4" name="search_text" style="margin-left:0px;" placeholder="<%=rb.getString("Title_SheBeiBianMa")%>" class="searchInputStyle omcLogInputSearch" onfocus="inputOnfocusStyle(this)" onblur="inputOnBlourStyle(this)">
                <div class="highQueryArrow">
                    <span class="highQueryTip" id="highQueryTip_4" onclick="moreQueryImgFun4()"><%=rb.getString("GaoJiChaXun")%></span>
                    <img class="queryHighBtnLog query-arrow" id="queryHighBtn_4" onclick="moreQueryImgFun4()" flag="1">
                </div>
            </div>
            <div id="isSuper">
                <b class="searchResultImgChangeStyle" onclick="vagueQuerySystemLogList()"></b>
            </div>
        </div> --%>
        <div id="omcSystemLogDiv" class="infoDetailStyle">        
	        <ul class="inputslist" >
	            <li>
	                <label style="display:inline-block;margin-bottom:5px;"><%=rb.getString("Title_SheBeiLeixing")%>:</label><br>
	                <select id="sys_deviceType" name="device_type" class="easyui-combobox border border-box device_type_4"></select>
	            </li>
	            <li>
	                <label for="sel_op_name_sys" style="display:inline-block;margin-bottom:5px;"><%=rb.getString("CaoZuoLeiXing")%>:</label><br>
	                <input id="sel_op_name_sys" name="operate_type" class="easyui-combobox border border-box sel_op_name_4" data-options="editable:false" style="height: 26px;width:200px;">
	            </li>
	            <li>
	                <label style="display:inline-block;margin-bottom:5px;"><%=rb.getString("KaiShiShiJian")%>:</label><br>
	                <input name="op_start_time" class="easyui-datetimebox border border-box op_start_time_4" data-options="editable:false" style="height: 26px;width:200px;">
	            </li>
	            <li>
	                <label style="display:inline-block;margin-bottom:5px;"><%=rb.getString("JieShuShiJian")%>:</label><br>
	                <input name="op_end_time" class="easyui-datetimebox border border-box op_end_time_4" data-options="editable:false" style="height: 26px;width:200px;">
	            </li>
	        </ul>
	        <div class="windowButtonGroup" style="margin-left:18px;margin-bottom:10px;float:left">
	        	<a href="#" class="linkbutton linkbutton_trend" onclick="accurateQuerySystemLogList()" style="float:left;"><span><%=rb.getString("ChaXun")%></span></a>
	        	<a href="#" class="linkbutton linkbutton_nowanna" style="float:left;" onclick="resetSystemLogQueryInput()"><span style="min-width:auto;padding:0 20px"><%=rb.getString("ChaXunChongZhi")%></span></a>
	        </div>
	        
        </div>
    </form>
</div>

<%-- 导出日志文件 --%>
<form id="formExportLogData" style="display:none" method="post"></form>

 
<script type="text/javascript">
var operationLog_column = [];
var securityLog_column = [];

$(function(){
	closeLoading();
	
	//操作日志--初始化列 
    operationLog_column = [
    	{field:'operation_name',sortable:true,width:100,title:'<%=rb.getString("CaoZuoMingCheng")%>'},
    ];
     
	if( isCloud == 'true' && is_super_user == "true"){
		operationLog_column.push({field:'operator_code',sortable:true,width:100,title:'<%=rb.getString("YunYingShang")%>'});
	}
	operationLog_column.push({field:'user_code',sortable:true,width:100,title:'<%=rb.getString("YongHuMingCheng")%>'});
	operationLog_column.push({field:'operate_ip',sortable:true,width:100,title:'<%=rb.getString("CaoZuoIPAddress")%>'});
	operationLog_column.push({field:'op_start_time',sortable:true,width:100,title:'<%=rb.getString("ShiJian")%>'});
	operationLog_column.push({field:'detail',sortable:true,width:500,title:'<%=rb.getString("XiangXiJiLu")%>'});
	
	//操作日志--初始化表格  
	$("#tabelOperationNote").datagrid({
		url:'${ctx}/system/operationNote/getOperationNotePageList.action',
		queryParams:{timeZone:timeZone},
		fit:true,
		fitColumns:true,
		nowrap:false,
		border:false,
		striped:true,
		singleSelect:true,
		rownumbers:true,
		pagination:true,
		pagePosition:'bottom',
		toolbar:'#toolbar_operationNote',
		onLoadSuccess:gridOnLoadSuccessForAutoSize,
		columns:[operationLog_column]
	})
	
	//安全日志--初始化列    
    securityLog_column = [
    	{field:'operation_name',sortable:true,width:100,title:'<%=rb.getString("CaoZuoMingCheng")%>'},
    ];
     
	if( isCloud == 'true' && is_super_user == "true"){
		securityLog_column.push({field:'operator_code',sortable:true,width:100,title:'<%=rb.getString("YunYingShang")%>'});
	}
	securityLog_column.push({field:'user_code',sortable:true,width:100,title:'<%=rb.getString("YongHuMingCheng")%>'});
	securityLog_column.push({field:'operate_ip',sortable:true,width:100,title:'<%=rb.getString("CaoZuoIPAddress")%>'});
	securityLog_column.push({field:'op_start_time',sortable:true,width:100,title:'<%=rb.getString("ShiJian")%>'});
	securityLog_column.push({field:'detail',sortable:true,width:500,title:'<%=rb.getString("XiangXiJiLu")%>'});
		
	//安全日志--初始化表格 
	$("#omcSecurityLogGrid").datagrid({
		url:'${ctx}/system/logmange/security/getSecurityLogPageList.action',
		queryParams:{timeZone:timeZone},
		fit:true,
		fitColumns:true,
		nowrap:false,
		border:false,
		striped:true,
		singleSelect:true,
		rownumbers:true,
		pagination:true,
		pagePosition:'bottom',
		toolbar:'#toolbar_omcSecurityLogGrid',
		onLoadSuccess:gridOnLoadSuccessForAutoSize,
		columns:[securityLog_column]
	})
	
		
	// 当为Cloud版，且管理员登录时，才显示高级查询中的运营商下拉框  
	if( isCloud == 'true' && is_super_user == "true"){
		$(".operatorItem").show();
	}else {
		$(".operatorItem").hide();
	}
	
	// 操作日志查询面板--填充"操作名称"下拉框
    $("#form_omcOperationLog #sel_op_name_op").combobox({
        url : "${ctx}/system/operationNote/getOperationNameForCombobox.action",
        valueField : 'id',
        textField : 'text',
        editable : false,
        panelHeight : 210
    });
	
    // 操作日志 -- 填充"运营商"下拉框
    $("#form_omcOperationLog #sel_operator_name_op").combobox({
        url : "${ctx}/system/operator/getOperatorListForCombobox.action",
        valueField : 'id',
        textField : 'text',
        editable : true,
        panelHeight : 210,
        onChange : function(newValue, oldValue){
        	$("#form_omcOperationLog #sel_user_code_op").combobox("reload",{operator_code:newValue});
        }
    });
 
    // 操作日志查询面板--填充"用户名称"下拉框
    $("#form_omcOperationLog #sel_user_code_op").combobox({
        url : "${ctx}/system/sysuser/getUserListForCombobox.action",
        valueField : 'id',
        textField : 'text',
        editable : true,
        panelHeight : 210
    });

    // 安全日志查询面板 -- 填充"操作名称"下拉框
    $("#form_omcSecurityLog  #sel_op_name_sec").combobox({
         url : "${ctx}/system/logmange/security/getOperationNameForCombobox.action",
         valueField : 'id',
         textField : 'text',
         editable : false,
         panelHeight : 210
    });
    
    // 安全日志  -- 填充"运营商"下拉框
    $("#form_omcSecurityLog #sel_operator_name_sec").combobox({
        url : "${ctx}/system/operator/getOperatorListForCombobox.action",
        valueField : 'id',
        textField : 'text',
        editable : true,
        panelHeight : 210,
        onChange : function(newValue, oldValue){
            $("#form_omcSecurityLog #sel_user_code_sec").combobox("reload",{operator_code:newValue});
        }
    });
    
    // 安全日志查询面板 -- 填充"用户名称"下拉框
    $("#form_omcSecurityLog #sel_user_code_sec").combobox({
         url : "${ctx}/system/sysuser/getUserListForCombobox.action",
         valueField : 'id',
         textField : 'text',
         editable : true,
         panelHeight : 210
    });

    // 系统日志查询面板 -- 填充"操作类型"下拉框
    $("#form_omcSystemLog  #sel_op_name_sys").combobox({
        url : "${ctx}/system/logmange/system/getOperationNameForCombobox.action",
        valueField : 'id',
        textField : 'text',
        editable : false,
        panelHeight : 210
    });
    
 	// 系统日志查询面板 -- 填充"设备类型"下拉框
    $("#form_omcSystemLog  #sys_deviceType").combobox({
        valueField : 'value',
        textField : 'text',
        editable : false,
        width:200,
        height:26,
        panelHeight : 210,
        data:[
        	{ value:'',text:'<%=rb.getString("QuanBu")%>'},
        	{ value:'eNB',text:'eNB'},
        ],
        loadFilter:function(data){
        	var obj={};
        	if(isSupportCPE == 'true'){
        		obj.value = 'CPE';
            	obj.text = 'CPE';
            	data.splice(2,2,obj);
        	}
        	
        	return data;
        }
    });

	if ("${isElfCell}" == "1") {
		$("#tableCellUpgradeRecordDB").datagrid({
			queryParams : {
				timeZone : timeZone
			}
		});
	}
	
	$("#operateName_1").bind("keyup", function(e) {
		if (e.keyCode == 13) {
			vagueQueryOperList();
		}
	});
	$("#sel_sn").bind("keyup", function(e) {
		if (e.keyCode == 13) {
			vagueQueryUpgradeHistoryList();
		}
	});
	$("#operateName_3").bind("keyup", function(e) {
		if (e.keyCode == 13) {
			vagueQuerySecurityLogList();
		}
	});
	$("#operateName_4").bind("keyup", function(e) {
		if (e.keyCode == 13) {
			vagueQuerySystemLogList();
		}
	});
	
	/* $(".queryHighBtnLog").each(function(){
		$(this).hover(function(){
			$(this).prev().show();
		},function(){
			if($("#operateName_1").is(":focus")){
				return;
			}
			if($("#operationNoteDiv").is(":visible")){
				return;
			}
			if($("#sel_sn").is(":focus")){
				return;
			}
			if($("#tableCellUpgradeNoteDiv").is(":visible")){
				return;
			}
			if($("#operateName_3").is(":focus")){
				return;
			}
			if($("#omcSecurityLogGridDiv").is(":visible")){
				return;
			}
			if($("#operateName_4").is(":focus")){
				return;
			}
			if($("#omcSystemLogDiv").is(":visible")){
				return;
			}
			$(this).prev().hide();
		});
	}) */
	
	/* $("#highQueryTip_1").mouseleave(function(){
		if($("#operateName_1").is(":focus")){
			return;
		}
		if($("#operationNoteDiv").is(":visible")){
			return;
		}
		$("#highQueryTip_1").hide();
	});
	
	$("#highQueryTip_2").mouseleave(function(){
		if($("#sel_sn").is(":focus")){
			return;
		}
		if($("#tableCellUpgradeNoteDiv").is(":visible")){
			return;
		}
		$("#highQueryTip_2").hide();
	});
	
	$("#highQueryTip_3").mouseleave(function(){
		if($("#operateName_3").is(":focus")){
			return;
		}
		if($("#omcSecurityLogGridDiv").is(":visible")){
			return;
		}
		$("#highQueryTip_3").hide();
	});
	
	$("#highQueryTip_4").mouseleave(function(){
		if($("#operateName_4").is(":focus")){
			return;
		}
		if($("#omcSystemLogDiv").is(":visible")){
			return;
		}
		$("#highQueryTip_4").hide();
	});
	
	$("#operateName_1").focus(function(){
		$("#highQueryTip_1").show();
	});
	
	$("#sel_sn").focus(function(){
		$("#highQueryTip_2").show();
	});
	
	$("#operateName_3").focus(function(){
		$("#highQueryTip_3").show();
	});
	
	$("#operateName_4").focus(function(){
		$("#highQueryTip_4").show();
	});
	
	$(document).click(function(e){	
		if($(e.target).closest(".window-mask").length==0&&$(e.target).closest(".messager-window").length==0&&$(e.target).closest(".infoDetailStyle").length==0
		   &&$(e.target).closest(".combo-p").length==0&&$(e.target).closest(".highQueryGroup").length==0&&$(e.target).closest(".searchResultImgChangeStyle").length==0
		   &&$(e.target).closest(".operateName").length==0&&$(e.target).closest(".highQueryArrow").length==0&&$(e.target).closest(".omcLogInputSearch").length==0&&$(e.target).closest(".calendar-other-month").length==0){
			 if($("#operationNoteDiv").is(":visible")){				 
			 	$("#operationNoteDiv").slideUp(500);
			 }else{
			 	$("#operationNoteDiv").slideUp();				 
			 }
			 if($("#tableCellUpgradeNoteDiv").is(":visible")){
				 $("#tableCellUpgradeNoteDiv").slideUp(500);
			 }else{
				 $("#tableCellUpgradeNoteDiv").slideUp();				 
			 }
			 if($("#omcSecurityLogGridDiv").is(":visible")){
				 $("#omcSecurityLogGridDiv").slideUp(500);
			 }else{
				 $("#omcSecurityLogGridDiv").slideUp();				 
			 }
			 if($("#omcSystemLogDiv").is(":visible")){
				 $("#omcSystemLogDiv").slideUp(500);
			 }else{
				 $("#omcSystemLogDiv").slideUp();				 
			 }
			 $("#queryHighBtn_1").removeClass('expanded');
			 $("#queryHighBtn_1").attr("flag","1"); 
			 $("#queryHighBtn_2").removeClass('expanded');
			 $("#queryHighBtn_2").attr("flag","1"); 
			 $("#queryHighBtn_3").removeClass('expanded');
			 $("#queryHighBtn_3").attr("flag","1"); 
			 $("#queryHighBtn_4").removeClass('expanded');
			 $("#queryHighBtn_4").attr("flag","1"); 
			 $("#highQueryTip_1").hide();
			 $("#highQueryTip_2").hide();
			 $("#highQueryTip_3").hide();
			 $("#highQueryTip_4").hide();
		 }		
	});  */
	// 权限控制
	setTimeout(function(){
		$('#omcLogTabsDiv span[tabtit]:visible').each(function(n,item){
			if(n==0) $(item).click();
		});
	},0);
	
	$('#omcSystemLogGrid').datagrid();
});
// 操作日志 点击出现高级查询弹窗
function moreQueryImgFun1(){
	if($("#queryHighBtn_1").attr("flag")=="1"){
		$("#operationNoteDiv").slideDown(500);
		$("#queryHighBtn_1").attr("flag","0");
		$("#queryHighBtn_1").addClass('expanded');
	}else{
		$("#operationNoteDiv").slideUp(500);
		$("#queryHighBtn_1").attr("flag","1");
		$("#queryHighBtn_1").removeClass('expanded');
	}
}
// 基站升级和自配置日志 点击出现高级查询弹窗
function moreQueryImgFun2(){
	if($("#queryHighBtn_2").attr("flag")=="1"){
		$("#tableCellUpgradeNoteDiv").slideDown(500);
		$("#queryHighBtn_2").attr("flag","0");
		$("#queryHighBtn_2").addClass('expanded');
	}else{
		$("#tableCellUpgradeNoteDiv").slideUp(500);
		$("#queryHighBtn_2").attr("flag","1");
		$("#queryHighBtn_2").removeClass('expanded');
	}
}
// 安全日志 点击出现高级查询弹窗
function moreQueryImgFun3(){
	if($("#queryHighBtn_3").attr("flag")=="1"){
		$("#omcSecurityLogGridDiv").slideDown(500);
		$("#queryHighBtn_3").attr("flag","0");
		$("#queryHighBtn_3").addClass('expanded');
	}else{
		$("#omcSecurityLogGridDiv").slideUp(500);
		$("#queryHighBtn_3").attr("flag","1");
		$("#queryHighBtn_3").removeClass('expanded');
	}
}
// 系统日志 点击出现高级查询弹窗
function moreQueryImgFun4(){
	if($("#queryHighBtn_4").attr("flag")=="1"){
		$("#omcSystemLogDiv").slideDown(500);
		$("#queryHighBtn_4").attr("flag","0");
		$("#queryHighBtn_4").addClass('expanded');
	}else{
		$("#omcSystemLogDiv").slideUp(500);
		$("#queryHighBtn_4").attr("flag","1");
		$("#queryHighBtn_4").removeClass('expanded');	
	}
}
/**
* 验证开始时间和结束时间是否合法
* @param startTimeStr{string}   开始时间
* @param endTimeStr{string}   结束时间
*/ 
function validateStartAndStopTime(startTimeStr, endTimeStr){
    if (isNotNull(startTimeStr) && isNotNull(endTimeStr)) {
        var startDate = dateParser(startTimeStr);
        var endDate = dateParser(endTimeStr);
        if (startDate.getTime() < endDate.getTime()) {
            return "true";
        }
    }
    if (!isNotNull(startTimeStr) && isNotNull(endTimeStr)) {
        return "true";
    }
    if (isNotNull(startTimeStr) && !isNotNull(endTimeStr)) {
        return "true";
    }
    if(""== startTimeStr&& "" == endTimeStr){
    	return "true";
    }
    return "false";
}
/**
* 判断时间是否为 空/ null
* @param arg{string}   时间
*/ 
function isNotNull(arg){
	if(arg == null){
		return false;
	}
	if(arg == ""){
		return false;
	}
	return true;
}
// 操作日志高级查询 重置
function resetoperationNoteQueryInput(){
 	$(".sel_op_name_1").combobox('setValue','');
 	$(".sel_ope_name_1").combobox('setValue','');
	$(".sel_user_code_1").combobox('setValue','');
	$(".sel_operate_ip_1").val(null);
	$(".sel_detail_1").val(null);
	$(".sel_op_start_time_1").datetimebox('setValue', null);
	$(".sel_op_end_time_1").datetimebox('setValue', null);
	
}
// 基站升级和自配置日志高级查询 重置
function resetUpgradeNoteQueryInput(){
	$(".sel_ip_addr_2").val(null);
	$(".sel_mode_name_2").combobox('setValue','0');
	$(".sel_ori_version_2").val(null);
	$(".sel_dest_version_2").val(null);
	$(".sel_op_start_time_2").datetimebox('setValue', null);
	$(".sel_op_end_time_2").datetimebox('setValue', null);
	
}
// 安全日志高级查询 重置
function resetSecurityLogQueryInput(){
	$(".sel_op_name_3").combobox('setValue','');
	$(".sel_ope_name_3 ").combobox('setValue','');
	$(".user_code_3").combobox('setValue','');
	$(".operate_ip_3").val(null);
	$(".detail_3").val(null);
	$(".op_start_time_3").datetimebox('setValue', null);
	$(".op_end_time_3").datetimebox('setValue', null);	
}
// 系统日志高级查询 重置
function resetSystemLogQueryInput(){
	$(".device_type_4").combobox('setValue','');
	$(".sel_op_name_4").combobox('setValue','');
	$(".op_start_time_4").datetimebox('setValue', null);
	$(".op_end_time_4").datetimebox('setValue', null);
}
//选择导出的日志类型   
function exportLogsTurn() {
    var choseType = "";
    $(".omcLogLists > span").each(function(index,ele){
        if($(this).hasClass("active")){
            choseType = $(this).attr("logtype");
        }
    })
    if (choseType == "op"){ //操作日志 
    	exportOperationLogs();
    } else if (choseType == "up"){ //设备升级日志 
        exportCellUpgradeConfigLogs();
    } else if (choseType == "sec"){ //安全日志 
    	exportSecurityLogs();
    } else if (choseType == "sys"){ //系统日志 
    	exportSystemLogs()
    }
} 

// 查询操作日志列表
function vagueQueryOperList() {
 	$(".sel_op_name_1").combobox('setValue','');
 	$(".sel_ope_name_1").combobox('setValue','');
	$(".sel_user_code_1").combobox('setValue','');
	$(".sel_operate_ip_1").val(null);
	$(".sel_detail_1").val(null);
	$(".sel_op_start_time_1").datetimebox('setValue', null);
	$(".sel_op_end_time_1").datetimebox('setValue', null);
	var params = $('#form_omcOperationLog').serializeJson();
	params.timeZone = timeZone;
	$("#tabelOperationNote").datagrid("load",params);
	$("#operationNoteDiv").slideUp(500);
	$(".queryHighBtnLog").removeClass('expanded');
	$(".queryHighBtnLog").attr("flag","1"); 
	$("#highQueryTip_1").hide();
}
// 高级查询操作日志列表
function accurateQueryOperList() {
	$("#operateName_1").val(null);
	var dataStartTime = $(".sel_op_start_time_1").datetimebox("getValue");
	var dataEndTime = $(".sel_op_end_time_1").datetimebox("getValue");
    //开始时间不能晚于结束时间
    var validTimeResult = validateStartAndStopTime(dataStartTime, dataEndTime);
    if ("false" == validTimeResult) {
    	showMsg('prompt_msg','<%=rb.getString("JieShuShiJianBuNengXiaoYuKaiShiShiJian")%>');
        return;
    }
	var params = $('#form_omcOperationLog').serializeJson();
	params.timeZone = timeZone;
	$("#tabelOperationNote").datagrid("load",params);
	$("#operationNoteDiv").slideUp(500);
	$(".queryHighBtnLog").removeClass('expanded');
	$(".queryHighBtnLog").attr("flag","1"); 
	$("#highQueryTip_1").hide();
}

// 按选择的格式导出操作日志
function exportOperationLogs(){
    // 将当前的查询参数赋值给form
    var data = $("#tabelOperationNote").datagrid("getData");
    if (data.rows.length > 0) {
    	var queryMap = $('#form_omcOperationLog').serializeJson();
        /* $("#formExportLogData").form('submit', {
            url: "${ctx}/system/operationNote/exportLogCsvFile.action",
            onSubmit: function(param){
                param.timeZone = timeZone;
                param.search_text = queryMap.search_text;
                param.operate_type = queryMap.op_id;
                param.operator_code = queryMap.operator_code;
                param.user_code = queryMap.user_code;
                param.operate_ip = queryMap.operate_ip;
                param.detail = queryMap.detail;
                param.start_time = queryMap.op_start_time;
                param.end_time = queryMap.op_end_time;
    			var bool = checkParams(param)
    			if(!bool) return false;
            }
        }); */
        exportByForm("${ctx}/system/operationNote/exportLogCsvFile.action",{
        	timeZone: timeZone,
            search_text: queryMap.search_text,
            operate_type: queryMap.op_id,
            operator_code: queryMap.operator_code,
            user_code: queryMap.user_code,
            operate_ip: queryMap.operate_ip,
            detail: queryMap.detail,
            start_time: queryMap.op_start_time,
            end_time: queryMap.op_end_time
        });
    } else {
    	showMsg('prompt_msg','<%=rb.getString("MeiYouYaoDaoChuDeShuJu")%>');
        return;
    }
}

//查询基站升级配置历史列表
function vagueQueryUpgradeHistoryList() {
	$(".sel_ip_addr_2").val(null);
	$(".sel_mode_name_2").combobox('setValue','0');
	$(".sel_ori_version_2").val(null);
	$(".sel_dest_version_2").val(null);
	$(".sel_op_start_time_2").datetimebox('setValue', null);
	$(".sel_op_end_time_2").datetimebox('setValue', null);
	var params = $('#form_tableCellUpgradeNote').serializeJson();
	params.timeZone = timeZone;
	$("#tableCellUpgradeRecordDB").datagrid("load",params);
	$("#tableCellUpgradeNoteDiv").slideUp(500);
	$(".queryHighBtnLog").removeClass('expanded');
	$(".queryHighBtnLog").attr("flag","1"); 
	$("#highQueryTip_2").hide();
}
// 基站升级和自配置日志高级查询 重置
function accurateQueryUpgradeHistoryList() {
	$("#sel_sn").val(null);
	var dataStartTime = $(".sel_op_start_time_2").datetimebox("getValue");
	var dataEndTime = $(".sel_op_end_time_2").datetimebox("getValue");
    //开始时间不能晚于结束时间
    var validTimeResult = validateStartAndStopTime(dataStartTime, dataEndTime);
    if ("false" == validTimeResult) {
    	showMsg('prompt_msg','<%=rb.getString("JieShuShiJianBuNengXiaoYuKaiShiShiJian")%>');
        return;
    }
	var params = $('#form_tableCellUpgradeNote').serializeJson();
	params.timeZone = timeZone;
	$("#tableCellUpgradeRecordDB").datagrid("load",params);
	$("#tableCellUpgradeNoteDiv").slideUp(500);
	$(".queryHighBtnLog").removeClass('expanded');
	$(".queryHighBtnLog").attr("flag","1"); 
	$("#highQueryTip_2").hide();
}

// 导出基站升级配置历史列表
function exportCellUpgradeConfigLogs(){
	// 将当前的查询参数赋值给form
	var queryMap = $('#form_tableCellUpgradeNote').serializeJson();
	$("#upgrade_form_timeZone").val(timeZone);
	$("#upgrade_form_sn").val(queryMap.sn);
	$("#upgrade_form_ip_addr").val(queryMap.ip_addr);
	$("#upgrade_form_task_name").val(queryMap.task_name);
	$("#upgrade_form_mode_name").val(queryMap.mode_name);
	$("#upgrade_form_ori_version").val(queryMap.ori_version);
	$("#upgrade_form_dest_version").val(queryMap.dest_version);
	$("#upgrade_form_op_start_time").val(queryMap.upgrade_op_start_time);
	$("#upgrade_form_op_end_time").val(queryMap.upgrade_op_end_time);
	$("#upgrade_form_result").val(queryMap.upgrade_result);
	
	var data = $("#tableCellUpgradeRecordDB").datagrid("getData");
	if (data.rows.length > 0) {
		var url = "${ctx}/system/operationNote/exportCellUpgradeAndConfigData.action";
		/* $("#formExportCellUpgradeLogData").form('submit', {
			url: url,
			onSubmit: function(param){
				var bool = checkParams(param)
				if(!bool) return false;
			}
		}); */
		exportByForm(url,$("#formExportCellUpgradeLogData").serializeJSON());
	} else {
		showMsg('prompt_msg','<%=rb.getString("MeiYouYaoDaoChuDeShuJu")%>');
		return;
	}
}

 // 基站升级历史-所属模块内容处理
function moduleFormatter(value, rowData, rowIndex){
	if (value == null) {
		return null;
	} else if (value == "1") {
		value = "<%= rb.getString("ShengJiCeLue")%>";
	} else if (value == "2") {
		value = "<%= rb.getString("SoftwareZiDongShengJi")%>";
	} else if (value == "3") {
		value = "<%= rb.getString("ParamZiDongPeiZhi")%>";
	}
	return value;
}

//查询安全日志操作列表
function vagueQuerySecurityLogList() {
	$(".sel_op_name_3").combobox('setValue','');
	$(".sel_ope_name_3 ").combobox('setValue','');
	$(".user_code_3").combobox('setValue','');
	$(".operate_ip_3").val(null);
	$(".detail_3").val(null);
	$(".op_start_time_3").datetimebox('setValue', null);
	$(".op_end_time_3").datetimebox('setValue', null);	
    var params = $('#form_omcSecurityLog').serializeJson();
    params.timeZone = timeZone;
    $("#omcSecurityLogGrid").datagrid("load",params);
    $("#omcSecurityLogGridDiv").slideUp(500);
	$(".queryHighBtnLog").removeClass('expanded');
	$(".queryHighBtnLog").attr("flag","1"); 
	$("#highQueryTip_3").hide();
}
//高级查询安全日志操作列表
function accurateQuerySecurityLogList() {
	$("#operateName_3").val(null);
	var dataStartTime = $(".op_start_time_3").datetimebox("getValue");
	var dataEndTime = $(".op_end_time_3").datetimebox("getValue");
	//开始时间不能晚于结束时间
	var validTimeResult = validateStartAndStopTime(dataStartTime, dataEndTime);
	if ("false" == validTimeResult) {
		showMsg('prompt_msg','<%=rb.getString("JieShuShiJianBuNengXiaoYuKaiShiShiJian")%>');
	    return;
	}
    var params = $('#form_omcSecurityLog').serializeJson();
    params.timeZone = timeZone;
    $("#omcSecurityLogGrid").datagrid("load",params);
    $("#omcSecurityLogGridDiv").slideUp(500);
	$(".queryHighBtnLog").removeClass('expanded');
	$(".queryHighBtnLog").attr("flag","1"); 
	$("#highQueryTip_3").hide();
}

// 导出安全操作日志CSV文件
function exportSecurityLogs(){
    // 将当前的查询参数赋值给form
    var data = $("#omcSecurityLogGrid").datagrid("getData");
    if (data.rows.length > 0) {
    	var queryMap = $('#form_omcSecurityLog').serializeJson();
    	/* $("#formExportLogData").form('submit', {
            url: "${ctx}/system/logmange/security/exportLogCsvFile.action",
            onSubmit: function(param){
                param.timeZone = timeZone;
                param.search_text = queryMap.search_text;
                param.operate_type = queryMap.op_id;
                param.operator_code = queryMap.operator_code;
                param.user_code = queryMap.user_code;
                param.operate_ip = queryMap.operate_ip;
                param.detail = queryMap.detail;
                param.start_time = queryMap.op_start_time;
                param.end_time = queryMap.op_end_time;
    			var bool = checkParams(param)
    			if(!bool) return false;
            }
        }); */
        exportByForm("${ctx}/system/logmange/security/exportLogCsvFile.action",{
        	timeZone: timeZone,
            search_text: queryMap.search_text,
            operate_type: queryMap.op_id,
            operator_code: queryMap.operator_code,
            user_code: queryMap.user_code,
            operate_ip: queryMap.operate_ip,
            detail: queryMap.detail,
            start_time: queryMap.op_start_time,
            end_time: queryMap.op_end_time
        });
    } else {
    	showMsg('prompt_msg','<%=rb.getString("MeiYouYaoDaoChuDeShuJu")%>');
        return;
    }
}

//查询系统日志操作列表
function vagueQuerySystemLogList() {
	$(".device_type_4").combobox('setValue','');
	$(".sel_op_name_4").combobox('setValue','');
	$(".op_start_time_4").datetimebox('setValue', null);
	$(".op_end_time_4").datetimebox('setValue', null);
    var params = $('#form_omcSystemLog').serializeJson();
    params.timeZone = timeZone;
    $("#omcSystemLogGrid").datagrid("load",params);
    $("#omcSystemLogDiv").slideUp(500);
	$(".queryHighBtnLog").removeClass('expanded');
	$(".queryHighBtnLog").attr("flag","1"); 
	$("#highQueryTip_4").hide();
}
//高级查询系统日志操作列表
function accurateQuerySystemLogList() {
	$("#operateName_4").val(null);
	var dataStartTime = $(".op_start_time_4").datetimebox("getValue");
	var dataEndTime = $(".op_end_time_4").datetimebox("getValue");
	//开始时间不能晚于结束时间
	var validTimeResult = validateStartAndStopTime(dataStartTime, dataEndTime);
	if ("false" == validTimeResult) {
		showMsg('prompt_msg','<%=rb.getString("JieShuShiJianBuNengXiaoYuKaiShiShiJian")%>');
	    return;
	}
    var params = $('#form_omcSystemLog').serializeJson();
    params.timeZone = timeZone;
    $("#omcSystemLogGrid").datagrid("load",params);
    $("#omcSystemLogDiv").slideUp(500);
	$(".queryHighBtnLog").removeClass('expanded');
	$(".queryHighBtnLog").attr("flag","1"); 
	$("#highQueryTip_4").hide();
}

// 导出系统操作日志CSV文件
function exportSystemLogs(){
    // 将当前的查询参数赋值给form
    var data = $("#omcSystemLogGrid").datagrid("getData");
    if (data.rows.length > 0) {
    	var queryMap = $('#form_omcSystemLog').serializeJson();
        /* $("#formExportLogData").form('submit', {
            url: "${ctx}/system/logmange/system/exportLogCsvFile.action",
            onSubmit: function(param){
                param.timeZone = timeZone;
                param.search_text = queryMap.search_text;
                param.device_type = queryMap.device_type;
                param.operate_type = queryMap.operate_type;
                param.start_time = queryMap.op_start_time;
                param.end_time = queryMap.op_end_time;
    			var bool = checkParams(param)
    			if(!bool) return false;
            }
        }); */
        exportByForm("${ctx}/system/logmange/system/exportLogCsvFile.action",{
            timeZone: timeZone,
            search_text: queryMap.search_text,
            device_type: queryMap.device_type,
            operate_type: queryMap.operate_type,
            start_time: queryMap.op_start_time,
            end_time: queryMap.op_end_time
        });
    } else {
    	showMsg('prompt_msg','<%=rb.getString("MeiYouYaoDaoChuDeShuJu")%>');
        return;
    }
}
</script>