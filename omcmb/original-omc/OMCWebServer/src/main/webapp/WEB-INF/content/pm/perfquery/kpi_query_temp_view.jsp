<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp"%>

<!-- 查看模板 -->
<div class="slidebarTitleDiv">
	<div id="viewTemplateTitle" style="display:inline-block;"><%=rb.getString("XinXi")%></div>
	<a class="el-icon el-icon-close slideIcon" onclick="closeViewTemplateDiv()" style="position: absolute; top:5px; right: 20px;font-size: 30px;"></a>
</div>
<div class="slideBody">
	<div class="slideCont">
		<div class="splitGroup">
			<div class="splitGroup_title"><%=rb.getString("JiBenXinXi")%></div>
			<div class="splitGroup_body">			
				<div style="height:55px;">					
					<label class="inputTittleCss"><%=rb.getString("MuBanMingCheng")%></label>
					<input id="viewTemplateName" class="inputDivCss border border-box" disabled="true" style="width: 400px;"/>
				</div>
				<div>					
					<label class="inputTittleCss"><%=rb.getString("MiaoShu")%></label>
					<textarea id="viewTemplateDescription" class="inputDivCss border border-box" disabled="true" style="width:800px;height:130px;vertical-align:top;resize:none;"></textarea>
				</div>		
			</div>    		
		</div>
		<div class="splitGroup">
			<div class="splitGroup_title"><%=rb.getString("ZhouQiSheDing")%></div>
	   		<div class="splitGroup_body period_readonly" style="border: 1px solid #E2E3E9;width: 850px;display:flex;">
				<div style="width: 140px;border-right: 1px solid #E9E9E9;background:#FDFDFD;">
					<div class="periodHeader" style="text-align: center;"><%=rb.getString("ChaXunLiDu")%></div>
					<div id="viewTempPeriod" class="flex-ctn">
						<span class="period-item">
							<input type="radio" name="period" value="15" checked/><label> 15Min</label>
						</span>
						<span class="period-item">
							<input type="radio" name="period" value="60"/><label> 60Min</label>
						</span>
						<span class="period-item">
							<input type="radio" name="period" value="1440"/><label> 24hour</label>
						</span>
					</div>
					<label class="inputTipCss errorTipStyle"></label>
				</div>
				<div style="flex-grow:1;">
					<div id="viewKpiTimeSlotDiv" class="periodHeader">
						<label style="margin:0 50px 0 20px;"><%=rb.getString("ChaXunShiDuan")%></label>
						<input type="radio" name="timeSlot" id="allTimeSlot" checked value="1"><label for="allTimeSlot"><%=rb.getString("SuoYouShiDuan")%></label>
						<input type="radio" name="timeSlot" id="partTimeSlot" value="2" class="not_support"><label class="not_support" for="partTimeSlot"><%=rb.getString("ZhiDingShiDuan")%></label>
					</div>
					
					<div id="viewTimeSlotSelectDiv" style="padding:20px;" class="readonly not_support">
						<div>
							<span class="timeSlotTitle"><%=rb.getString("Zhou")%><%=rb.getString("MaoHao")%></span>
							<div class="selectRegion_week">
								<p value="0">Sun</p>
								<p value="1">Mon</p>
								<p value="2">Tues</p>
								<p value="3">Wed</p>
								<p value="4">Thurs</p>
								<p value="5">Fri</p>
								<p value="6">Sat</p>
							</div>
						</div>
						<div style="margin-top:20px;">
							<span class="timeSlotTitle"><%=rb.getString("XiaoShi")%><%=rb.getString("MaoHao")%></span>
							<div class="selectRegion_hour">
								<p value="0">0</p>
								<p value="1">1</p>
								<p value="2">2</p>
								<p value="3">3</p>
								<p value="4">4</p>
								<p value="5">5</p>
								<p value="6">6</p>
								<p value="7">7</p>
								<p value="8">8</p>
								<p value="9">9</p>
								<p value="10">10</p>
								<p value="11">11</p>
								<p value="12">12</p>
								<p value="13">13</p>
								<p value="14">14</p>
								<p value="15">15</p>
								<p value="16">16</p>
								<p value="17">17</p>
								<p value="18">18</p>
								<p value="19">19</p>
								<p value="20">20</p>
								<p value="21">21</p>
								<p value="22">22</p>
								<p value="23">23</p>
							</div>
						</div>
					</div>
				</div>
			</div>
	   	</div>
		<div class="splitGroup" id="group_device_div">
			<div class="splitGroup_title"><%=rb.getString("SheBeiLieBiao")%></div>
			<div style="display: flex;flex-direction: column;">
				<div style="margin-left: 54px;margin-top: 20px;">
					<span><%=rb.getString("SheBeiXuanZheFangShi")%></span>
				</div>
				<div id="viewTempDeviceType" style="display: flex;flex-direction:column;width: 90%;border: 1px solid #E9E9E9;margin-left: 54px;margin-top:5px" >
					<div>
						<span class="period-item" style="border:none;">
							<input type="radio" name="selectType" value="1" id="select_type_1" /><label for="select_type_1"><%=rb.getString("SheBeiZu")%></label>
						</span>
						<span class="period-item" style="border:none;">
							<input type="radio" name="selectType" value="2" id="select_type_2" /><label for="select_type_2"><%=rb.getString("KPISheBei")%></label>
						</span>
					</div>
				   <div class="" id="view_group_select_div" style="display: none;margin-top: 20px;padding:0 20px;">
						<div id="modify_temp_group_wrap" style="border:1px solid #E9E9E9; height: 350px;width:400px;">
							<table id='autoAccessDeviceGroupIdView'></table>
						</div>
				   </div>
				   <div class="" id="view_devices_select_div" style="margin-top: 20px;">
						<div style="margin-top: 0px;height:350px;padding:0 20px;">
							<table id="view_kpiTemplate_device_datagrid"></table>
						</div>
				   </div>
		   		</div>
   			</div>
   		</div>
	   	<div class="splitGroup">
	   		<div class="splitGroup_title"><%=rb.getString("ZhiBiaoXuanZe")%></div>
	  		<div style="width:90%;height:350px;margin-left:54px;border:1px solid #E9E9E9;">
	  			<table id="view_kpiTemplate_kpi_datagrid" ></table>
			</div>
		</div>
	</div>
</div>
<div class="slideFooter">
	<input type="checkbox" disabled value="true" id="temp_default_checkbox"/><label for="temp_default_checkbox"> <%=rb.getString("SheWeiMoRen")%></label>
</div>

<script type="text/javascript">
	var curViewEnbGnbEgwType = sessionStorage.getItem('viewTemplateEnbGnbOrEgw');
	
	$(function(){
	    closeLoading();
	    var datagrid = $("#kpiTemplateDatagrid").datagrid("getSelected");  
	    var params = {};
		params.tempId = datagrid.tempId;
		params.timeZone = timeZone;
	    getKPITemlateInfo(params);
	
	    /** 
		* 设备选择类型
		* $(this).val() == 1  设备组
		* $(this).val() == 2  设备
		**/
	    $('#viewTempDeviceType input').click(function(){
	    	var groupDiv = $("#view_group_select_div"),
	    		deviceDiv = $('#view_devices_select_div');
	    	
	    	if($(this).val() == 1){
	    		groupDiv.show();
	    		deviceDiv.hide();
	    	}else {
	    		groupDiv.hide();
	    		deviceDiv.show();
	    	}
	    	
	    	$("#autoAccessDeviceGroupIdView").datagrid('resize');
	    });
	    
	    var curDeviceGroupListUrl = '', curDeviceListUrl = '', curKpiListUrl ='', deviceColumnsList = [];
	    if(curViewEnbGnbEgwType == '0'){
	    	curDeviceGroupListUrl = '${ctx}/pm/template/getSelDeviceGroupList.action';
	    	curDeviceListUrl = '${ctx}/pm/template/viewTemplateRelDeviceList.action';
	    	curKpiListUrl = '${ctx}/pm/template/viewTemplateRelIndicatorsList.action';
	    	deviceColumnsList = [	 
	    		{field : 'groupName',width : 60,title: '<%=rb.getString("SheBeiZu")%>'}, 
				{field : 'smallCellCode',hidden : true}, 
				{field : 'serialNumber',sortable : true,width : 100,title : '<%=rb.getString("XiaoZhanBianMa")%>'}, 
				{field: 'hostName',sortable: true,width: 120,title: '<%=rb.getString("HostName")%>'},
				{field: 'product',sortable: true,width: 120,title: '<%=rb.getString("ChanPinLeiXing")%>'}
            ], 
	    	kpiColumnsList = [	 
	    		{field:"catagoryName",width : 70,title: '<%=rb.getString("ZhiBiaoGongNengJi")%>'}, 
	            {field : "kpiId",sortable : true,title : '<%=rb.getString("ZhiBiaoID")%>',width : 60}, 
		        {field:"kpiName",sortable : true,title: '<%=rb.getString("ZhiBiaoMingCheng")%>',width:200},
	            {field: 'product_type',sortable: false,width: 120,title: '<%=rb.getString("ChanPinLeiXing")%>'}
            ]
	    }else if(curViewEnbGnbEgwType == '1'){
	    	curDeviceGroupListUrl = '${ctx}/gnb/pm/template/getSelDeviceGroupList.action';
	    	curDeviceListUrl = '${ctx}/gnb/pm/template/viewTemplateRelDeviceList.action';
	    	curKpiListUrl = '${ctx}/gnb/pm/template/viewTemplateRelIndicatorsList.action';
	    	deviceColumnsList = [	 
	    		{field : 'groupName',width : 60,title: '<%=rb.getString("SheBeiZu")%>'}, 
				{field : 'smallCellCode',hidden : true}, 
				{field : 'serialNumber',sortable : true,width : 100,title : '<%=rb.getString("XiaoZhanBianMa")%>'}, 
				{field: 'hostName',sortable: true,width: 120,title: '<%=rb.getString("HostName")%>'}
            ],
	    	kpiColumnsList = [	 
	    		{field:"catagoryName",width : 70,title: '<%=rb.getString("ZhiBiaoGongNengJi")%>'}, 
	            {field : "kpiId",sortable : true,title : '<%=rb.getString("ZhiBiaoID")%>',width : 60}, 
		        {field:"kpiName",sortable : true,title: '<%=rb.getString("ZhiBiaoMingCheng")%>',width:200}
            ] 
	    }else{
	    	curDeviceGroupListUrl = '${ctx}/egw/pm/template/getSelDeviceGroupList.action';
	    	curDeviceListUrl = '${ctx}/egw/pm/template/viewTemplateRelDeviceList.action';
	    	curKpiListUrl = '${ctx}/egw/pm/template/viewTemplateRelIndicatorsList.action';
	    	deviceColumnsList = [	 
	    		{field : 'groupName',width : 60,title: '<%=rb.getString("SheBeiZu")%>'}, 
				{field : 'smallCellCode',hidden : true}, 
				{field : 'serialNumber',sortable : true,width : 100,title : '<%=rb.getString("eGWBianMa")%>'}, 
				{field: 'hostName',sortable: true,width: 120,title: '<%=rb.getString("eGWMingCheng")%>'}
            ],
	    	kpiColumnsList = [	 
	    		{field:"catagoryName",width : 70,title: '<%=rb.getString("ZhiBiaoGongNengJi")%>'}, 
	            {field : "kpiId",sortable : true,title : '<%=rb.getString("ZhiBiaoID")%>',width : 60}, 
		        {field:"kpiName",sortable : true,title: '<%=rb.getString("ZhiBiaoMingCheng")%>',width:200}
            ]
	    }
	    
	  	//设备选择方式-设备组
		$("#autoAccessDeviceGroupIdView").datagrid({
			url:curDeviceGroupListUrl,
			queryParams : {
				"tempId":datagrid.tempId
			},
			singleSelect:false,
			fit:true,
			fitColumns:true,
			border:false,
			rownumbers:true,
			striped: true,
			idField:"id",
			onLoadSuccess : datagridLoadSuccess,
			columns: [[
				{field: 'id',hidden:true},
				{field: 'built_in',hidden:true},
				{field: 'group_name',width:100,title:'<%=rb.getString("SheBeiZuMingCheng") %>'}
			]],
		});
	  	
		//设备选择方式-设备
	    $("#view_kpiTemplate_device_datagrid").datagrid({
	    	url : curDeviceListUrl,
	    	queryParams:{
	    		tempId: datagrid.tempId
	    	},
			border : false,
			fit : true,
			fitColumns : true,
			rownumbers : true,
			striped : true,
			singleSelect : true,
			pageList : [ 50, 100, 150, 200, 250, 300 ],
			pagination : true,
			pagePosition : 'bottom',
			idField : 'smallCellCode',
			onLoadSuccess : datagridLoadSuccess,
			columns : [ deviceColumnsList]
		});
	    
		//指标选择
	    $('#view_kpiTemplate_kpi_datagrid').datagrid({
	    	url : curKpiListUrl,
	    	queryParams:{
	    		tempId: datagrid.tempId
	    	},
	        border:false,
	        fit:true,
	        fitColumns:true,
	        rownumbers : true,
	        striped : true,
	        singleSelect : true,
	        idField: 'kpiId',
			pageList : [ 50, 100, 150, 200, 250, 300 ],
	        pagination: true,
	        pagePosition: 'bottom',
	        onLoadSuccess : datagridLoadSuccess,
	  	    columns : [kpiColumnsList]
	  	 });
	   
	});
	
	/** 
	* 模板信息渲染
	* @param params[string] 选中行的模板 id
	**/
	function getKPITemlateInfo(params){
		var curTemplateInfoUrl = '';
		
	    if(curViewEnbGnbEgwType == '0'){
	    	curTemplateInfoUrl = '${ctx}/pm/template/getTemplateInfo.action';	    	
	    }else if(curViewEnbGnbEgwType == '1'){
	    	curTemplateInfoUrl = '${ctx}/gnb/pm/template/getTemplateInfo.action';
	    }else{
	    	curTemplateInfoUrl = '${ctx}/egw/pm/template/getTemplateInfo.action';	
	    }
	    
		$.post(curTemplateInfoUrl, params, function(data) {
			 if(data){
				 $("#viewTemplateName").val(data.tempName);			 
				 $("#viewTemplateDescription").val(data.description);
				 $("#viewTempPeriod").find('input[value='+data.reportPeriod+']').prop('checked',true);
				 
				 if(data.isAllOperator == 1) {
					 $('#group_device_div').hide();
				 }
				 if(data.is_default == "true") {
					// $("#temp_default_checkbox").prop('checked',true);
					$('input[type="checkbox"]').prop('checked',true); //ok
				 }
				 if(data.reportPeriod == "1440"){
					 $("input[value='1']").attr("checked",true);
					 $(".not_support").css("visibility",'hidden');
				 }else{
					 var week = data.week.split(",");
					 var hour = data.hour.split(",");
					 if(week[0] != "" && hour[0] != ""){
						 $("input[value='2']").attr("checked",true);
						 
						 //选中天
						 $.each($("#viewTimeSlotSelectDiv .selectRegion_week p"),function(index,ele){
							 if($.inArray($(this).attr("value"),week)>-1) {
								 $(this).addClass("active");
		 	                 }
						 })
						 //选中小时点 
						 $.each($("#viewTimeSlotSelectDiv .selectRegion_hour p"),function(index,ele){
							 if($.inArray($(this).html(),hour)>-1) {
								 $(this).addClass("active");
		 	                 }
						 })
					 }else{
						 $("#viewKpiTimeSlotDiv input[value='1']").attr("checked",true);
					 }
				 }
				 
				 $('#viewTempDeviceType input[value="'+data.selDeviceType+'"]').click();
				 $('#viewTempDeviceType input').prop("disabled",true);
				 
			 }else {
				 showMsg('error_msg',data["message"])
			 }
		 },"json");
	}
</script>