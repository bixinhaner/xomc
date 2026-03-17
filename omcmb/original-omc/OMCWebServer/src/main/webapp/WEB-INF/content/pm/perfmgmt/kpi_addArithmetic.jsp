<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.ck-radio-cls {
		min-width: 135px;
		display: flex;
		align-items: center;
		border: 1px solid #DEDFE6;
		border-radius: 5px;
		padding: 4px 8px;
		margin-right: 15px;
	}
	.ck-radio-cls label {
		margin-left: 5px;
		cursor: pointer;
	}
</style>
<div class="overflow-cls" style="flex-direction: column;">
<div class="slidebarTitleDiv" style="min-width: 1500px;position: relative;">
	<span><%=rb.getString("XinJianZhiBiao")%></span>
	<div class="circleIcon" style="right:15px;">
		<span class="el-icon el-icon-circle-close" onclick="kpiAddCancel()"></span>
		<div class="titleButtonText"><%=rb.getString("GuanBi")%></div>
	</div>
</div>
<div id="kpiManaAdd_body" class="slideBody" style="min-width: 1500px;">
	<div class="slideCont">
		<div class="splitGroup" style="width: 100%;">
			<div class="splitGroup_title"><%=rb.getString("JiBenXinXi")%></div>
	  		<div class="splitGroup_body form-group-inline">	
				<div class="CODE_PERFORMANCE_ADD_COUNTER hidden" style="width: 80%;">
					<label class="inputTittleCss"><%=rb.getString("Type")%></label>
					<div style="display: flex;align-items: center;padding: 2px 5px 26px 0px;">
						<span class="ck-radio-cls">
							<input type="radio" name="indicatorType" id="kpiTypeValue_add" oldValue="" checked value="kpi"/> 
							<label for="kpiTypeValue_add">Customize KPI</label>
						</span>
						<span class="ck-radio-cls">
							<input type="radio" name="indicatorType" id="kpiTypeValue1_add" oldValue="" value="counter"/> 
							<label for="kpiTypeValue1_add">Customize Counter</label>
						</span>
					</div>
				</div>
				<div class="enbPlmnLevelShow">
					<label class="inputTittleCss"><%=rb.getString("DengJi")%></label>
					<div style="display: flex;align-items: center;padding: 2px 5px 26px 0px;">
						<span class="ck-radio-cls">
							<input type="radio" name="indicatorLevel" id="levelType_enb" oldValue="" checked value="device"/> 
							<label for="levelType_enb">Device</label>
						</span>
						<span class="ck-radio-cls">
							<input type="radio" name="indicatorLevel" id="levelType_plmn" oldValue="" value="plmn"/> 
							<label for="levelType_plmn">PLMN</label>
						</span>
					</div>
				</div>
				<div>		
					<label class="inputTittleCss"><%=rb.getString("ZhiBiaoMingCheng")%></label>
					<input id="kpiNameValue_add" class="inputDivCss border border-box" oldValue="" maxLength="50"/>
					<label class="inputTipCss errorTipStyle"></label>
				</div> 			
				<div id="kpi_cus_name_div" style="display: none;">		
					<label class="inputTittleCss">Custom Name</label>
					<input id="kpiCustomNameValue_add" class="inputDivCss border border-box" oldValue="" maxLength="50"/>
					<label class="inputTipCss errorTipStyle"></label>
				</div> 

				<!--enb-->
				<div class='eNBFunctionSetShow'>					
					<label class="inputTittleCss"><%=rb.getString("SuoShuGongNengJi")%></label>
					<input id="funcSetValueId_add" name="funcSetValueName" class="easyui-combotree inputDivCss border border-box" style="height:26px;" oldValue=""/>
					<div class="operationDiv status_star"></div>
					<label id="funcSetValueIdTittle" class="inputTipCss errorTipStyle"></label>
				</div>
				<!--gnb,wcg-->
				<div class='functionSetShow'>					
					<label class="inputTittleCss"><%=rb.getString("SuoShuGongNengJi")%></label>
					<select id="funcSetValueId_add" name="funcSetValueName" class="easyui-combobox inputDivCss border border-box" style="height:26px;" oldValue=""></select>
					<div class="operationDiv status_star"></div>
					<label id="funcSetValueIdTittle" class="inputTipCss errorTipStyle"></label>
				</div>	

				<div>					
					<label class="inputTittleCss"><%=rb.getString("DanWei")%></label>
					<input id="kpiIdUnitValueId" name="kpiIdUnitValueName" class="easyui-combobox inputDivCss border border-box" style="height:26px;" oldValue=""/>
					<div class="operationDiv status_star"></div>
					<label class="inputTipCss errorTipStyle" id="kpiIdUnitValueIdTittle"></label>
				</div>	
				<div>					
					<label class="inputTittleCss"><%=rb.getString("TongJiLeiXing")%></label>
					<select id="StatisticalSetValueId" name="StatisticalSetValueName" class="easyui-combobox inputDivCss border border-box" style="height:26px;" oldValue=""></select>
					<label id="StatisticalSetValueIdTittle" class="inputTipCss errorTipStyle"></label>
				</div>
				<div class='eNBTypeShow'>
					<label class="inputTittleCss"><%=rb.getString("CeLiangNew")%></label>
					<input id="enableValueId" name="enableValueName" class="easyui-combobox inputDivCss border border-box" style="height:26px;" oldValue=""/>
					<div class="operationDiv status_star"></div>
					<label class="inputTipCss errorTipStyle" id="enableValueIdTittle"></label>
				</div>
				<div>	
	  				<div class="inputTittleCss"><%=rb.getString("ShuoMing")%></div>
	  				<textarea id="explainValue" class="InfoDivContent" maxLength="2000" style="display:block;height:100px;width:765px;padding:15px 0px 10px 15px;overflow-y:auto;resize:none;"></textarea>
					<label class="inputTipCss errorTipStyle"></label>
  				</div>
  			</div>
		</div>
		<div id="kpi_param_part_add" class="splitGroup" style="width: 100%;">
			<div class="splitGroup_title"><%=rb.getString("JiSuanGongShi")%></div>
	  		<div class="splitGroup_body">				
				<div style="margin-top:30px; position: relative;">
					<textarea id="calcExpValue" style="display:none;"></textarea>
					<div id="calcExpValueShow" class="InfoDivContent" style="display:block;height:100px;width:90%;padding:15px 0px 10px 15px;overflow-y:auto;" readonly="readonly"></div>
					<div tabindex="0" class="ivu-tag" style="display:none" id="hiddenTag">
					    <span class="ivu-tag-text" contenteditable="true"> </span>
					    <i class="ivu-icon ivu-icon-ios-close-empty titleIcon_close"></i>
					</div>
					<div id="productAll" class="InfoDivContent" style="display:none;"></div>					
				</div>
	            <div style="margin-top:10px;width:90%;">
		            <div class="kpiSymbol" style="float:initial;">					            
		                <a class="linkbutton linkbutton_nowanna" onclick="addSign('+')"><span>+</span></a>
		                <a class="linkbutton linkbutton_nowanna" onclick="addSign('-')"><span>-</span></a>
		                <a class="linkbutton linkbutton_nowanna" onclick="addSign('*')"><span>*</span></a>
		                <a class="linkbutton linkbutton_nowanna" onclick="addSign('/')"><span>/</span></a>
		                <a class="linkbutton linkbutton_nowanna" onclick="addSign('(')"><span>(</span></a>
		                <a class="linkbutton linkbutton_nowanna" onclick="addSign(')')"><span>)</span></a>
		                <a class="linkbutton linkbutton_nowanna" onclick="addSign('num')"><span>0-9</span></a>
		                <a class="linkbutton linkbutton_nowanna" onclick="addSign('Duration')"><span style="padding:0 10px;">Duration</span></a>
		                <a class="linkbutton linkbutton_nowanna" onclick="addSign('clear')"><span><%=rb.getString("QingChu")%></span></a>
		            </div>
		            <label id="calcExpValueShowTitle" class="inputTipCss errorTipStyle" ></label>
	            </div>
	            <div id="kpiAlgorithmicOperNameDiv" style="padding: 10px;border: 1px solid #DEDFE6;width:90%;margin: 10px 0 30px 0;"></div>
	            <div style="width:90%;">
					<div style="display: flex;">
						<div class="queryGroup" style="margin-left: 0;">
							<input id="kpi_search_text" style="width:300px;" placeholder="<%=rb.getString("ZhiBiaoMingChengZhiBiaoJi")%>" />
							<b class="el-icon el-icon-common-search" onclick="reloadKpiTree('add')"></b>
						</div>
						<input id="productTypeSelect" type="text" class="border border-box file_info required" maxlength=100 style="width: 200px;height:32px; border-radius: 4px; margin: 1px 0 0 20px;display: none" />    
					</div>
					<div id="kpiAlgorithmicDiv" style="height:450px;">
				        <div class="leftCol" style="margin-left:0px;">
							<div class="infoDivTitle"><%=rb.getString("ZhiBiaoGongNengJi")%></div>
				        	<div class="InfoDivContent" style="overflow-x:hidden" id="kpiSetTree"></div>
				        </div>
				        <div class="rightCol" style="width:870px" >
				        	<div class="infoDivTitle"><%=rb.getString("XingNengZhiBiao")%></div>
							<div class="InfoDivContent">
								<table id="kpiListDatagrid"></table>
							</div>
				        </div>
					</div>
				</div>
			</div>
		</div>
   </div>
</div>
<div class="slideFooter" style="min-width: 1500px;">
     <a href="#" class="linkbutton linkbutton_trend addKpi" onclick="addKPIArithCommit()" ><span><%=rb.getString("QueDing")%></span></a>
     <a href="#" class="linkbutton linkbutton_nowanna" onclick="kpiAddCancel()"><span><%=rb.getString("QuXiao")%></span></a>
</div>
</div>
<script>
//网元标识
var addKpiNetType = kpiManagePageVue.currentKpiNetType && kpiManagePageVue.currentKpiNetType != '' ? kpiManagePageVue.currentKpiNetType : sysMain.headType;

var searchTextKPIZhiBiao = '';
var searchProductTypeSelect = '';
var curProduct = '';
//level 新建指标的标识
var addOrModifyKpiLevel = '';
//选中功能集的 device_type，用于过滤 kpiSetTree 数据
var selectedFuncSetDeviceType = '';
$(function(){
	// 初始化产品类型下拉列表
	console.log(addKpiNetType,'...addKpiNetType')
	if( addKpiNetType == 'enb'){
		$('.eNBTypeShow').show();
		$('.eNBFunctionSetShow').show();
		$('.functionSetShow').hide();
		//Level 显示
		$('.enbPlmnLevelShow').show();
		//设备 产品类型下拉选择
		axios.post('${ctx}/cell/cpeinfos/getEnbMonitorProductList.action?isGnb=0&isAll=1').then(function(response){
			var data = response.data;
			// 动态删除 BTS
			data = data.filter(function(item) {
				return item !== 'BTS';
			});
			
			if(data.length == 0){
				searchProductTypeSelect = 'no';
	   		}else{
	   			curProduct =  'ALL,' + data.join(',');
	   			searchProductTypeSelect = curProduct;
	   		}
			
			var arr = [{name:'<%=rb.getString("QuanBu")%>',value:''}];
			data.map(function(item){
				if (item){
					arr.push({name:item,value:item})
				}
			})
			
			$("#productTypeSelect").combobox({
				editable:false,
		    	textField:'name',
		    	valueField:'value',
		    	data: arr,
		    	onSelect:function(){
		    		searchProductTypeSelect =  $('#productTypeSelect').combobox('getValue');
		    		if(searchProductTypeSelect == ''){
		    			searchProductTypeSelect = curProduct;
		    		} 
		    		reloadKpiTree('add')
		    	},
		    }); 
			//绑定类型事件
			$('#kpiManaAdd_body [name=indicatorLevel]').on('change',function(ev){
				var LevelType = ev.target.value;
				addOrModifyKpiLevel = LevelType;
				addSign('clear');
				
				reloadKpiTree('add')
			})
			//初始化指标算法选择树
			loadFuncSet('add');
		}).catch(function(error){})
	}else{
		//初始化指标算法选择树
		loadFuncSet('add');
		$('.eNBTypeShow').hide();
		$('.eNBFunctionSetShow').hide();
		$('.functionSetShow').show();
		//Level 隐藏
		$('.enbPlmnLevelShow').hide();
	}
	//绑定类型事件
	$('#kpiManaAdd_body [name=indicatorType]').on('change',function(ev){
		var indicatorType = ev.target.value;

		if(indicatorType != 'counter') {
			$('#kpi_param_part_add').show();
			$('#kpi_cus_name_div').hide();
		}else {
			$('#kpi_param_part_add').hide();
			$('#kpi_cus_name_div').show();
		}
	})
	//初始化指标功能集下拉列表
    initFuncSetValue('funcSetValueId_add');
    //初始化指标单位下拉列表
    initKPIUnit('kpiIdUnitValueId');
    //初始化统计类型下拉列表
    initStatisticSetValue('StatisticalSetValueId');
    //初始化启用下拉列表
    if(addKpiNetType == 'enb'){
        initenableSetValue('enableValueId','','add');
    }
    
    //清空KPI门限数据
	 $("#generalColor").css('background','#CCCC66');
	 $("#seriousColor").css('background','#CC6666');
	 $("[name=generalColor]").val('#CCCC66');
	 $("[name=seriousColor]").val('#CC6666'); 
	/*  阻止冒泡  */
	$('#kpiManagePage .chose').click(function(event){
	    event.stopPropagation()      
	})
	$("#kpi_search_text").bind("keyup", function (event) {
        if (event.keyCode == 13) {
        	reloadKpiTree('add');
        }
    });
	// 指标名称 失去焦点验证
	$("#kpiNameValue_add").blur(function(){
		if($(this).val().trim().length>0){
			$(this).next().html("");
		}else{
			$(this).next().html("<%=rb.getString("ZhiBiaoMingChengWeiKong")%>");
		}
	}) 
	//指标功能集 选择事件
	if(addKpiNetType == 'enb'){
		$("#funcSetValueId_add").combotree({
			onSelect:function(node){
				// 清空错误提示
				/*$("#funcSetValueIdTittle").html("");
				
				// 清空计算公式数据
				$("#calcExpValueShow").html("");
				$("#calcExpValue").val("");
				
				// 清空计算公式错误提示
				$("#calcExpValueShowTitle").html("");*/

			addSign('clear');
			
			// 保存选中节点的 device_type，用于过滤 kpiSetTree
			selectedFuncSetDeviceType = node.device_type || '';
			console.log('选中功能集的 device_type:', selectedFuncSetDeviceType);
			
			reloadKpiTree('add');
			console.log('功能集切换，已清空计算公式数据', node);				// 构建完整路径显示（父级/子级）
				var tree = $("#funcSetValueId_add").combotree('tree');
				var parentNode = tree.tree('getParent', node.target);
				
				var displayText = node.text;
				if(parentNode){
					displayText = parentNode.text + '/' + node.text;
				}
				
				console.log('设置显示文本:', displayText);
				
				// 延迟设置，确保在 combotree 内部更新完成之后执行
				setTimeout(function(){
					// 方法1: 使用 setText API
					$("#funcSetValueId_add").combotree('setText', displayText);
					
					// 方法2: 直接操作 DOM - 最可靠的方式
					setTimeout(function(){
						// 查找实际的输入框并强制设置值
						var $textInput = $("#funcSetValueId_add").next('.combo').find('input.combo-text');
						if($textInput.length > 0){
							$textInput.val(displayText);
							console.log('成功通过DOM设置:', displayText);
						} else {
							// 备用方案：查找所有输入框
							$('input.combo-text').each(function(){
								var $input = $(this);
								// 如果输入框的值是节点文本，说明是我们要找的
								if($input.val() === node.text){
									$input.val(displayText);
									console.log('成功通过备用方案设置:', displayText);
									return false; // 跳出循环
								}
							});
						}
					}, 50);
				}, 10);
			}
		})
	}else{
		
	}
	
	/* 切换统计类型时，清空上一次的错误提示信息  */
	$("#StatisticalSetValueId").combobox({
		onChange:function(){
			$("#calcExpValueShowTitle").html("");
		}
	})
	$("#enableValueId").combobox({
		onChange:function(){
			$("#enableValueIdTittle").html("");
		}
	})
	document.querySelector('#calcExpValueShow').addEventListener('click',function(event){
		if(event.target.tagName != 'INPUT'){
			var checkedInput = this.querySelector('.pointer.checked');
			if(checkedInput) checkedInput.className = 'pointer';
		}
	});
})
//确认 修改基础指标 KPI 基本信息及门限等
function addKPIArithCommit() {
	var isBlank=true;
	var threadData ={};
	var arithmetic = $("#calcExpValueShow").val();
	
	//类型：kpi / counter
	var indicatorType = $('#kpiManaAdd_body [name=indicatorType]:checked').val();

    //指标功能集 
	if(addKpiNetType == 'enb'){
		var funcSetValue =  $("#funcSetValueId_add").combotree("getValue");
	}else{
		var funcSetValue =  $("#funcSetValueId_add").combobox("getValue");
	}
  	//指标单位  
    var kpiIdUnitValue =  $("#kpiIdUnitValueId").combobox("getValue");
  	//统计类型
  	var statisticValue =  $("#StatisticalSetValueId").combobox("getValue");
    //是否启用
  	var isEnableValue =  $("#enableValueId").combobox("getValue");
  	//解释 
    var explainValue = $("#explainValue").val().replace(/\n/g, " ");
    //计算公式
	var kpiDatagrid_arith = $('#kpiDatagrid').datagrid('getSelected');
	//已选指标所对应的交集
	var productOurAll = $('#productAll').val();
	
	var calcExpValue = $("#calcExpValue").val();
	if(indicatorType != 'counter') {// 非counter类型校验表达式
		if (calcExpValue.length == 0) {
			$("#calcExpValueShowTitle").html("<%=rb.getString("JiSuanGongShiWeiKong")%>");
			$("#kpiManaAdd_body").animate({scrollTop:$("#calcExpValueShow").offset().top+530},0);
			isBlank=false;
		}
		
		if(statisticValue == 'pct'){
			if(calcExpValue.indexOf('/') < 0){
				$("#calcExpValueShowTitle").html("<%=rb.getString("GongShiBiXuBaoHanChuFa")%>");
				$("#kpiManaAdd_body").animate({scrollTop:$("#calcExpValueShow").offset().top+230},0);
				isBlank=false;
			}
		}
	}

	//指标名称 
    var kpiNameValue_add = $("#kpiNameValue_add").val().trim();
    if (kpiNameValue_add.length == 0) {
    	$("#kpiNameValue_add").next().html("<%=rb.getString("ZhiBiaoMingChengWeiKong")%>");
    	$("#kpiNameValue_add").focus();
		isBlank=false;
    }
	// 自定义名称
	var kpiCustomNameValue_add = $("#kpiCustomNameValue_add").val().trim();

    if(addKpiNetType == 'enb'){
	    if(intersetionTip == true){
	    	showMsg('error_msg','<%=rb.getString("YiXuanZhiBiaoDeChanPinLeiXingBuYiZhi")%>');
	    	isBlank=false;
		}
    }
    if(!isBlank){
		return;
	}
    //指标门限数据-验证门限数据
 	if(!$("#kpiThresholdForm").form("validate")){
 		return;
 	}
    var threadFormData={};
    var threadArray=$("#kpiThresholdForm").serializeArray();
	 $.each(threadArray,function(index,obj){
		 if(obj.value==null||obj.value==""){
			threadFormData[obj.name]=null;
		 }else{
			threadFormData[obj.name]=obj.value;
		 }
	 })
	 if(JSON.stringify(threadFormData)!="{}"){
		if(threadFormData.generalBegin==null&&threadFormData.generalEnd!=null
		 	    ||threadFormData.generalBegin!=null&&threadFormData.generalEnd==null
		 	    ||threadFormData.seriousBegin==null&&threadFormData.seriousEnd!=null
		 	    ||threadFormData.seriousBegin!=null&&threadFormData.seriousEnd==null){
		 		showMsg('prompt_msg','<%=rb.getString("ZhiBiaoMenXianSheZhiFanWeiBuQuan")%>')
		           return false; 
		}
		threadFormData=JSON.stringify(threadFormData);
	}else{
		threadFormData=null;
	}
	 
	
    //封装参数
    var curSaveUrl = '',
    	params = {
    		"kpiName" : kpiNameValue_add,
    		"catagoryId": funcSetValue,
    		"unit":kpiIdUnitValue,
    		"statisType":statisticValue,
    		"definition": explainValue,
    		"arithmetic": calcExpValue
        };

	if(indicatorType == 'counter') {
		params = {
    		"kpiName" : kpiNameValue_add,
			"kpiCustomName": kpiCustomNameValue_add,
    		"catagoryId": funcSetValue,
    		"unit":kpiIdUnitValue,
    		"statisType":statisticValue,
    		"definition": explainValue,
			"indicatorType": indicatorType,
			"custName": $('#kpiCustomNameValue_add').val().trim()
        };
	}
    
    //提交请求
	if(addKpiNetType == 'enb'){
		//Level 类型：enb / plmn
		var levelType = $('#kpiManaAdd_body [name=indicatorLevel]:checked').val();
		
		params.product_type = productOurAll;
		params.isEnable = isEnableValue;
		params.indicatorLevel = levelType;
		curSaveUrl = '${ctx}/pm/indicatormg/addOrModifyIndicator.action';		
	}else if(addKpiNetType == 'gnb'){
		curSaveUrl = '${ctx}/gnb/pm/indicatormg/addOrModifyIndicator.action';
	}else{
		curSaveUrl = '${ctx}/egw/pm/indicatormg/addOrModifyIndicator.action';	
	}
	
    if(!$(".addKpi").hasClass("forbidden")){
		$(".addKpi").addClass("forbidden");
		$.post(curSaveUrl, params, function (data) {
	        if (data["success"]) {
	        	try{
	        		itemArr = new Array();
	        		nameArr = new Array();
	        	}catch(e){}
	        	$('#kpiDatagrid').datagrid('reload');
				$("#kpiArithmeticTree").tree("reload"); //左侧父级
	        	showMsg('success_msg','<%=rb.getString("ChengGong")%>');
	        	kpiAddCancel()
	        } else {
				$(".addKpi").removeClass("forbidden");
	        	showMsg('error_msg',data.message);
	        }
	    }, "json");
	}
}
</script>