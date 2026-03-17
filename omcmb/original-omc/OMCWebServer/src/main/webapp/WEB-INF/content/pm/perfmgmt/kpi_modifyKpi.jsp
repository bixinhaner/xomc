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
<div class="slidebarTitleDiv" style='position: relative;'>
	<div style="display:inline-block;"><%=rb.getString("XiuGaiZhiBiao")%></div>
	<div class="circleIcon" style="right:15px;">
		<span class="el-icon el-icon-circle-close" onclick="kpiModifyCancel()" style='font-size: 12px !important; line-height: 26px; float: unset !important; margin: 0 !important;position: unset !important;'></span>
	</div>
</div>
<div id="kpiManaModify_body" class="slideBody">
	<div class="slideCont">
		<div class="splitGroup">
			<div class="splitGroup_title"><%=rb.getString("JiBenXinXi")%></div>
	  		<div class="splitGroup_body form-group-inline">		
				<!-- 等级选择: 设备 | PLMN -->
				 <div class="modifyEnbPlmnLevelShow">
					<label class="inputTittleCss"><%=rb.getString("DengJi")%></label>
					<div style="display: flex;align-items: center;padding: 2px 5px 26px 0px;">
						<span class="ck-radio-cls">
							<input type="radio" name="modifyLevelType" id="levelTypeEnb_modify" oldValue="" checked value="device"/> 
							<label for="levelTypeEnb_modify">Device</label>
						</span>
						<span class="ck-radio-cls">
							<input type="radio" name="modifyLevelType" id="kpiTypePlmn_modify" oldValue="" value="plmn"/> 
							<label for="kpiTypePlmn_modify">PLMN</label>
						</span>
					</div>
				</div>		
				<div>					
					<label class="inputTittleCss"><%=rb.getString("ZhiBiaoMingCheng")%></label>
					<input id="kpiNameValue_modify" class="inputDivCss border border-box" oldValue="" maxLength="50"/>
					<label class="inputTipCss errorTipStyle"></label>
				</div>
				<!--enb/GSM-->
				<div class='eNBModifyFunctionSetShow'>					
					<label class="inputTittleCss"><%=rb.getString("SuoShuGongNengJi")%></label>
					<input id="funcSetValueId_modify" name="funcSetValueName" class="easyui-combotree inputDivCss border border-box" style="height:26px;" oldValue=""/>
					<label id="funcSetValueIdTittle" class="inputTipCss errorTipStyle"></label>
				</div>
				<!--gnb,wcg-->
				<div class='modifyFunctionSetShow'>					
					<label class="inputTittleCss"><%=rb.getString("SuoShuGongNengJi")%></label>
					<select id="funcSetValueId_modify" name="funcSetValueName" class="easyui-combobox inputDivCss border border-box" style="height:26px;" oldValue=""></select>
					<label id="funcSetValueIdTittle" class="inputTipCss errorTipStyle"></label>
				</div>

	  			<div>					
					<label class="inputTittleCss"><%=rb.getString("ZhiBiaoID")%></label>
					<input id="kpiIdValue" class="inputDivCss border border-box"  value="<%=rb.getString("ZiDongShengCheng")%>" disabled="true"/>
					<label class="inputTipCss errorTipStyle"></label>
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
					<div class="operationDiv status_star"></div>
					<label id="StatisticalSetValueIdTittle" class="inputTipCss errorTipStyle"></label>
				</div>
				<div class='eNBTypeShow'>
					<label class="inputTittleCss"><%=rb.getString("CeLiangNew")%></label>
					<select id="enableValueId" name="enableValueName" class="easyui-combobox inputDivCss border border-box" style="height:26px;" oldValue=""></select>
					<div class="operationDiv status_star"></div>
					<label id="enableValueIdTittle" class="inputTipCss errorTipStyle"></label>
				</div>
				<div style="width:90%;">	
	  				<div class="inputTittleCss"><%=rb.getString("ShuoMing")%></div>
	  				<textarea id="explainValue" class="InfoDivContent" maxLength="2000" style="display:block;height:100px;width:765px;padding:15px 0px 10px 15px;overflow-y:auto;resize:none;"></textarea>
					<label class="inputTipCss errorTipStyle"></label>
  				</div>
  			</div>
		</div>
		<div class="splitGroup">
			<div class="splitGroup_title"><%=rb.getString("JiSuanGongShi")%></div>
	  		<div class="splitGroup_body">	
				<div style="margin-top:30px; position: relative;">
					<textarea id="calcExpValue" style="display:none;"></textarea>
					<div id="calcExpValueShow" class="InfoDivContent" style="display:block;height:100px;width:765px;padding:15px 0px 10px 15px;overflow-y:auto;" readonly="readonly"></div>
					<div tabindex="0" class="ivu-tag" style="display:none" id="hiddenTag">
					    <span class="ivu-tag-text" contenteditable="true"> </span>
					    <i class="ivu-icon ivu-icon-ios-close-empty titleIcon_close"></i>
					</div>
					<div id="productAll" class="InfoDivContent" style="display:none;"></div>
				</div>
	            <div style="margin-top:10px;">
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
	            </div>
	            <div id="kpiAlgorithmicOperNameDiv" style="padding: 10px;border: 1px solid #DEDFE6;width:761px;margin-top: 10px;"></div>
	            <div style="margin-top:10px;">
	            	<label id="calcExpValueShowTitle" class="inputTipCss errorTipStyle" ></label>
	            </div>
	            <div>
	            	<input id="updateProductTypeSelect" type="text" class="border border-box file_info required" maxlength=100 style="width: 200px;height:26px; display:none;" />    
	            	<div class="queryGroup">
		            
						<input id="kpi_search_text" style="width:300px;" placeholder="<%=rb.getString("ZhiBiaoMingChengZhiBiaoJi")%>" />
						<b class="el-icon el-icon-common-search" onclick="reloadKpiTree()"></b>
					</div>
	            </div>
	            
	            <div id="kpiAlgorithmicDiv" style="height:430px;">
			        <div class="leftCol" style="margin-left:0px;">
						<div class="infoDivTitle"><%=rb.getString("ZhiBiaoGongNengJi")%></div>
			        	<div class="InfoDivContent" style="overflow-x:hidden" id="kpiSetTree"></div>
			        </div>
			        <div class="rightCol" style="width:980px" >
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
  
<div class="slideFooter">
    <span class="el-button el-button--primary modifyKpi" onclick="modifyKpiArithCommit()" ><%=rb.getString("QueDing")%></span>
    <span class="el-button" onclick="kpiModifyCancel()"><%=rb.getString("QuXiao")%></span>
</div>

<script>
var modifyKpiNetType = kpiManagePageVue.currentKpiNetType && kpiManagePageVue.currentKpiNetType != '' ? kpiManagePageVue.currentKpiNetType : sysMain.headType;

var searchTextKPIZhiBiao = '';
var addOrModifyKpiLevel = ''; // 新建页面的等级标识
var kpiId = "";
var oldKpiData = {};
if(curOnClickRowId){
	kpiId = curOnClickRowId;
}
itemArr = new Array();
nameArr = new Array();
$(function(){
  	 //指标门限数据清空
    $("#kpiThresholdForm").form("clear");
    //加载指标及门限信息
	var infoUrl = '';
	
    if(modifyKpiNetType == 'enb'){
    	$('.eNBTypeShow').show();
		$('.eNBModifyFunctionSetShow').show();
		$('.modifyFunctionSetShow').hide();
		//Level 显示
		$('.modifyEnbPlmnLevelShow').show();
    	infoUrl = '${ctx}/pm/indicatormg/getIndicatorInfo.action';
		
		//设备 产品类型下拉选择  4G 显示产品类型选择
		axios.post('${ctx}/cell/cpeinfos/getEnbMonitorProductList.action?isGnb=0&isAll=1').then(function(response){
			var data = response.data;
			// 动态删除 BTS
			data = data.filter(function(item) {
				return item !== 'BTS';
			});
			
			if(data.length == 0){
				searchProductTypeSelect = 'no';
	   		}else{
	   			curProduct = 'ALL,' + data.join(',');
	   			searchProductTypeSelect = curProduct;
	   		}
			var arr = [{name:'<%=rb.getString("QuanBu")%>',value:''}];
			data.map(function(item){
				if (item){
					arr.push({name:item,value:item})
				}
			})
			
			$("#updateProductTypeSelect").combobox({
				editable:false,
		    	textField:'name',
		    	valueField:'value',
		    	data: arr,
		    	onSelect:function(){
		    		searchProductTypeSelect =  $('#updateProductTypeSelect').combobox('getValue');
		    		if(searchProductTypeSelect == ''){
		    			searchProductTypeSelect = curProduct;
		    		} 
		    		reloadKpiTree()
		    	},
		    }); 

			//绑定类型事件
			$('#kpiManaModify_body [name=modifyLevelType]').on('change',function(ev){
				var LevelType = ev.target.value;
				
				addOrModifyKpiLevel = LevelType;
				addSign('clear');
				
				reloadKpiTree()
			})
			
		}).catch(function(error){})
	}else if(modifyKpiNetType == 'gnb'){
		infoUrl = '${ctx}/gnb/pm/indicatormg/getIndicatorInfo.action';
		$('.eNBTypeShow').hide();
		$('.eNBModifyFunctionSetShow').hide();
		$('.modifyFunctionSetShow').show();
		//Level 显示
		$('.modifyEnbPlmnLevelShow').hide();
	}else{
		infoUrl = '${ctx}/egw/pm/indicatormg/getIndicatorInfo.action';
		$('.eNBTypeShow').hide();
		$('.eNBModifyFunctionSetShow').hide();
		$('.modifyFunctionSetShow').show();
		//Level 显示
		$('.modifyEnbPlmnLevelShow').hide();
	}
	
    $.post(infoUrl, {"kpiId":kpiId}, function (data) {
		if(data){
			oldKpiData = data;
			//初始化指标功能集下拉列表
            initFuncSetValue('funcSetValueId_modify',data["catagoryId"]);
            //初始化指标单位下拉列表
            initKPIUnit('kpiIdUnitValueId', data["unit"]);
            //初始化统计类型下拉列表
            initStatisticSetValue('StatisticalSetValueId',data["statisType"]); 
          	//初始化启用下拉列表
          	if(modifyKpiNetType == 'enb'){
                initenableSetValue('enableValueId', data['isEnable']);
            }
            //初始化指标算法选择树
	        loadFuncSet('mofidy');
			//设置值
			$("#kpiNameValue_modify").val(data.kpiName);
			$("#kpiIdValue").val(data.kpiId);
			$("#explainValue").val(data.definition);
	         
	        $("#calcExpValue").val(data.arithmetic);
	        var calcExpValueShowName = (data.arithmetic);
			var params = data.arithmetic;
			
			if(modifyKpiNetType == 'enb'){
				$('#productAll').val(data.product_type);
				//产品类型赋值
				//var updateInfoProduct = $('#productAll').val();
				//修改指标时，获取的计算公式中的产品类型
				//params['product_type'] = updateInfoProduct.replace(/\//g,',');
				//回显 level
				if(data.indicatorLevel == 'plmn') {
					$('#kpiTypePlmn_modify').attr('checked',true);
				}else{
					$('#levelTypeEnb_modify').attr('checked',true);
				}
				addOrModifyKpiLevel = data.indicatorLevel;
			}else{
				//5g,egw 不需要产品类型
			}
			
		   formatRuleReview('#calcExpValueShow',params,updateCalcExpValue);
		   $("#generalColor").css('background',data.generalColor);
           $("#seriousColor").css('background',data.seriousColor);
           $("[name=generalColor]").val(data.generalColor);
		   $("[name=seriousColor]").val(data.seriousColor);
           $("#generalTimeLevel").val(data.generalTimeLevel);
           $("#seriousTimeLevel").val(data.seriousTimeLevel);
           if(data.generalBegin != "null"){
	           $("#generalBegin").val(data.generalBegin);
           }
           if(data.generalEnd != "null"){
	           $("#generalEnd").val(data.generalEnd);
           }
           if(data.seriousBegin != "null"){
	           $("#seriousBegin").val(data.seriousBegin);
           }
           if(data.seriousEnd != "null"){
	           $("#seriousEnd").val(data.seriousEnd);
           }
		}
	 }, "json");
	
	/*  阻止冒泡  */
	$('#kpiManagePage .chose').click(function(event){
	    event.stopPropagation()      
	})
	$("#kpi_search_text").bind("keyup", function (event) {
        if (event.keyCode == 13) {
        	reloadKpiTree();
        }
    });
	$("#kpiNameValue_modify").keyup(function(){
		if($(this).val().trim().length>0){
			$(this).next().html("");
		}else{
			$(this).next().html("<%=rb.getString("ZhiBiaoMingChengWeiKong")%>");
		}
	}) 
	
	if(modifyKpiNetType == 'enb'){
		$("#funcSetValueId_modify").combotree({onChange:function(){
				$("#funcSetValueIdTittle").html("");
			}
		})
	}else {
		$("#funcSetValueId_modify").combobox({onChange:function(){
				$("#funcSetValueIdTittle").html("");
			}
		})
	}
	
	/* 切换统计类型时，清空上一次的错误提示信息  */
	$("#StatisticalSetValueId").combobox({onChange:function(){
			$("#calcExpValueShowTitle").html("");
		}
	})
	$("#enableValueId").combobox({onChange:function(){
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
function modifyKpiArithCommit() {
	var isBlank=true;
	var kpi_thread_data ={};
	var arithmetic = $("#calcExpValueShow").val();
	 //已选指标所对应的交集
	var productOurAll = $('#productAll').val();
    //指标功能集 
	if(modifyKpiNetType == 'enb'){
		var funcSetValue =  $("#funcSetValueId_modify").combotree("getValue");
	}else{
		var funcSetValue =  $("#funcSetValueId_modify").combobox("getValue");
	}
    
    if (funcSetValue.length === 0) {
    	$("#funcSetValueIdTittle").html("<%=rb.getString("ZhiBiaoGongNengJiWeiKong")%>");
    	$("#kpiManaModify_body").animate({scrollTop:$("#funcSetValueId_modify").offset().top},200);
		isBlank=false;
    }
  	//指标单位  
    var kpiIdUnitValue =  $("#kpiIdUnitValueId").combobox("getValue");
    if (kpiIdUnitValue.length === 0) {
    	$("#kpiIdUnitValueIdTittle").html("<%=rb.getString("DanWeiWeiKong")%>");
    	$("#kpiManaModify_body").animate({scrollTop:$("#kpiIdUnitValueId").offset().top+80},200);
		isBlank=false;
    }
    //统计类型  
    var statisTypeValue =  $("#StatisticalSetValueId").combobox("getValue");
    if (statisTypeValue.length === 0) {
    	$("#StatisticalSetValueIdTittle").html("<%=rb.getString("TongJiLeiXingWeiKong")%>");
    	$("#kpiManaModify_body").animate({scrollTop:$("#StatisticalSetValueId").offset().top+80},200);
		isBlank=false;
    }
  	//是否启用 
    var isEnableValue =  $("#enableValueId").combobox("getValue");
    /*if (isEnableValue.length === 0) {
    	$("#enableValueIdTittle").html("是否启用为空");
    	$("#kpiManaModify_body").animate({scrollTop:$("#enableValueId").offset().top+80},200);
		isBlank=false;
    }*/
  	//解释 
    var explainValue = $("#explainValue").val().replace(/\n/g, " ");
    //计算公式
	var kpiDatagrid_arith = $('#kpiDatagrid').datagrid('getSelected');
	var calcExpValue = $("#calcExpValue").val();
	if (calcExpValue.length == 0) {
    	$("#calcExpValueShowTitle").html("<%=rb.getString("JiSuanGongShiWeiKong")%>");
    	$("#kpiManaModify_body").animate({scrollTop:$("#calcExpValueShow").offset().top+530},0);
		isBlank=false;
    }
	
	if(statisTypeValue == 'pct'){
		if(calcExpValue.indexOf('/') < 0){
			$("#calcExpValueShowTitle").html("<%=rb.getString("GongShiBiXuBaoHanChuFa")%>");
			$("#kpiManaModify_body").animate({scrollTop:$("#calcExpValueShow").offset().top+230},0);
			isBlank=false;
		}
	}
	
	//指标名称 
    var kpiNameValue_modify = $("#kpiNameValue_modify").val().trim();
    if (kpiNameValue_modify.length == 0) {
    	$("#kpiNameValue_modify").next().html("<%=rb.getString("ZhiBiaoMingChengWeiKong")%>");
    	$("#kpiNameValue_modify").focus();
		isBlank=false;
    }
    if(modifyKpiNetType == 'enb'){
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
    var saveUrl = '',
    	params = {
    		"kpiId": kpiId,
    		"kpiName": kpiNameValue_modify,
    		"catagoryId": funcSetValue,
    		"unit": kpiIdUnitValue,
    		"statisType":statisTypeValue,
    		"definition": explainValue,
    		"arithmetic": calcExpValue
    		//"threadData": threadFormData
        };
    //提交请求
   if(!$(".modifyKpi").hasClass("forbidden")){
		if(modifyKpiNetType == 'enb'){
			//Level 类型：enb / plmn
			var levelType = $('.slideBody [name=modifyLevelType]:checked').val();
			params.indicatorLevel = levelType;
			params.product_type = productOurAll;
			params.isEnable = isEnableValue;
			saveUrl = '${ctx}/pm/indicatormg/addOrModifyIndicator.action';
			
			if(oldKpiData.kpiName == kpiNameValue_modify && oldKpiData.catagoryId == funcSetValue && oldKpiData.unit == kpiIdUnitValue && oldKpiData.statisType == statisTypeValue
			&& oldKpiData.isEnable == isEnableValue && oldKpiData.definition == explainValue && oldKpiData.arithmetic.keys.join('') == calcExpValue && oldKpiData.indicatorLevel == levelType){
				showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>')
				return;
		   	}
			$(".modifyKpi").addClass("forbidden");
		}else if(modifyKpiNetType == 'gnb'){
			saveUrl = '${ctx}/gnb/pm/indicatormg/addOrModifyIndicator.action';
			if(oldKpiData.kpiName == kpiNameValue_modify && oldKpiData.catagoryId == funcSetValue && oldKpiData.unit == kpiIdUnitValue && oldKpiData.statisType == statisTypeValue
		   	 && oldKpiData.definition == explainValue && oldKpiData.arithmetic.keys.join('') == calcExpValue){
				showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>')
				return;
		   	}
			$(".modifyKpi").addClass("forbidden");
		}else{
			saveUrl = '${ctx}/egw/pm/indicatormg/addOrModifyIndicator.action';
			if(oldKpiData.kpiName == kpiNameValue_modify && oldKpiData.catagoryId == funcSetValue && oldKpiData.unit == kpiIdUnitValue && oldKpiData.statisType == statisTypeValue
		   	 && oldKpiData.definition == explainValue && oldKpiData.arithmetic.keys.join('') == calcExpValue){
				showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>')
				return;
		   	}
			$(".modifyKpi").addClass("forbidden");
		}
		
		$.post(saveUrl, params, function (data) {
	        if (data["success"]) {
	        	try{
	        		itemArr = new Array();
	        		nameArr = new Array();
	        	}catch(e){}
	        	$('#kpiDatagrid').datagrid('reload');
				$("#kpiArithmeticTree").tree("reload"); //左侧父级
				showMsg('success_msg','<%=rb.getString("ChengGong")%>');
	            kpiModifyCancel();
	        } else {
				$(".modifyKpi").removeClass("forbidden");
	        	showMsg('error_msg',data["message"]);
	        }
	    }, "json");
	}
}
</script>