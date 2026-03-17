<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	.timeGranularityStyle { height: 25px; width: 90px; margin-left: 165px; background: #EAF1F4; }
	#kpiView_caculate_div .ivu-tag:hover { background: #e9fbff; }
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
<div class="slidebarTitleDiv el-card__header" style='position: relative;'>
	<div style="display:inline-block;border-bottom:2px solid;">
		<c:if test="${isBasic == 'true' }">
		<%=rb.getString("XiuGaiZhiBiao")%>
		</c:if>
		<c:if test="${isBasic != 'true' }">
		<%=rb.getString("XinXi")%>
		</c:if>
	</div>
	<div class="circleIcon" style="right:15px;">
		<span class="el-icon el-icon-circle-close" onclick="kpiModifyCancel()" style='font-size: 12px !important; line-height: 26px; float: unset !important; margin: 0 !important;position: unset !important;'></span>
	</div>
</div>
<div class="slideBody">
	<div class="slideCont">
		<div class="splitGroup">
			<div class="splitGroup_title"><%=rb.getString("JiBenXinXi")%></div>
	  		<div class="splitGroup_body form-group-inline">	
				<!-- 类型选择：kpi | counter -->
				<div style="width: 80%;display: none;">
					<label class="inputTittleCss"><%=rb.getString("Type")%></label>
					<div style="display: flex;align-items: center;padding: 2px 5px 26px 0px;">
						<span class="ck-radio-cls">
							<input type="radio" name="indicatorType" id="kpiTypeValue_modify" oldValue="" checked value="kpi"/> 
							<label for="kpiTypeValue_modify">Customize KPI</label>
						</span>
						<span class="ck-radio-cls">
							<input type="radio" name="indicatorType" id="kpiTypeValue1_modify" oldValue="" value="counter"/> 
							<label for="kpiTypeValue1_modify">Customize Counter</label>
						</span>
					</div>
				</div>	
				<!-- 等级选择: 设备 | PLMN 基础指标不可修改等级，需置灰 -->
				 <div class="modifyEnbPlmnLevelShow">
					<label class="inputTittleCss"><%=rb.getString("DengJi")%></label>
					<div style="display: flex;align-items: center;padding: 2px 5px 26px 0px;">
						<span class="ck-radio-cls">
							<input type="radio" name="modifyLevelType" id="levelTypeEnb_modify" disabled="true" oldValue="" checked value="device" /> 
							<label for="levelTypeEnb_modify">Device</label>
						</span>
						<span class="ck-radio-cls">
							<input type="radio" name="modifyLevelType" id="kpiTypePlmn_modify" disabled="true" oldValue="" value="plmn" /> 
							<label for="kpiTypePlmn_modify">PLMN</label>
						</span>
					</div>
				</div>	
				<div>					
					<label class="inputTittleCss"><%=rb.getString("ZhiBiaoMingCheng")%></label>
					<input id="kpiNameValue_view" class="inputDivCss border border-box" oldValue="" maxLength="200" disabled="true"/>
					<label class="inputTipCss errorTipStyle"></label>
				</div> 	
				<!--当 isCustomView != 'true' 时-基础指标的查看，显示该字段,当 isCustomView == 'true' 时，隐藏"自定义指标名称"字段-->		
				<c:if test="${isCustomView != 'true' }">
					<div>
						<label class="inputTittleCss"><%=rb.getString("ZiDingYiZhiBiaoMingChen")%></label>
						<input id="kpiCustomNameValue" class="inputDivCss border border-box" oldValue="" maxLength="200" <c:if test="${isBasic != 'true' }"> disabled="true"</c:if>/>
						<label class="inputTipCss errorTipStyle"></label>
					</div>
				</c:if>

				<!--enb/GSM-->
				<div class='eNBModifyFunctionSetShow'>					
					<label class="inputTittleCss"><%=rb.getString("SuoShuGongNengJi")%></label>
					<input id="funcSetValueId_modify" name="funcSetValueName" class="easyui-combotree inputDivCss border border-box" style="height:26px;" oldValue=""/>
					<label id="funcSetValueIdTittle" class="inputTipCss errorTipStyle"></label>
				</div>
				<!--gnb,wcg-->
				<div class='modifyFunctionSetShow'>					
					<label class="inputTittleCss"><%=rb.getString("SuoShuGongNengJi")%></label>
					<select id="funcSetValueId_modify" name="funcSetValueName" class="easyui-combobox inputDivCss border border-box" disabled="true" style="height:26px;" oldValue=""></select>
					<label id="funcSetValueIdTittle" class="inputTipCss errorTipStyle"></label>
				</div>

	  			<div>					
					<label class="inputTittleCss"><%=rb.getString("ZhiBiaoID")%></label>
					<input id="kpiIdValue" class="inputDivCss border border-box"  value="<%=rb.getString("ZiDongShengCheng")%>" disabled="true"/>
					<label class="inputTipCss errorTipStyle"></label>
				</div>  
				<div>					
					<label class="inputTittleCss"><%=rb.getString("DanWei")%></label>
					<input id="kpiIdUnitValueId" name="kpiIdUnitValueName" class="easyui-combobox inputDivCss border border-box" disabled="true" style="height:26px;" oldValue=""/>
					<label class="inputTipCss errorTipStyle" id="kpiIdUnitValueIdTittle"></label>
				</div>	
				<div>					
					<label class="inputTittleCss"><%=rb.getString("TongJiLeiXing")%></label>
					<select id="StatisticalSetValueId" name="StatisticalSetValueName" class="easyui-combobox inputDivCss border border-box" disabled="true" style="height:26px;" oldValue=""></select>
					<label id="StatisticalSetValueIdTittle" class="inputTipCss errorTipStyle"></label>
				</div>
				<div class='eNBTypeShow'>
					<label class="inputTittleCss"><%=rb.getString("CeLiangNew")%></label>
					<input id="enableValueId" name="enableValueName" class="easyui-combobox inputDivCss border border-box" style="height:26px;" oldValue="" <c:if test="${isBasic != 'true' }"> disabled="true"</c:if>/>
					<label class="inputTipCss errorTipStyle" id="enableValueIdTittle"></label>
				</div>
	  			<div style="width: 80%;">
	  				<div class="infoDivTitle"><%=rb.getString("ShuoMing")%></div>
	  				<textarea id="explainValue" class="InfoDivContent" style="display:block;height:100px;width:765px;padding:15px 0px 10px 15px;overflow-y:auto;resize:none;" disabled="true"></textarea>
	  			</div>
	  		</div>
		</div> 
		<div id="kpi_param_part_modify" class="splitGroup">
			<div class="splitGroup_title"><%=rb.getString("JiSuanGongShi")%></div>
	  		<div class="splitGroup_body">	
				<div>
					<textarea id="calcExpValue" style="display:none;"></textarea>
					<div id="calcExpValueShow" class="InfoDivContent" style="display:block;height:100px;width:90%;padding:15px 0px 10px 15px;overflow-y:auto;" readonly="readonly"></div>
					<div tabindex="0" class="ivu-tag" style="display:none" id="hiddenTag">
					    <span class="ivu-tag-text" contenteditable="true"> </span>
					</div>
				</div>
		        <div id="kpiAlgorithmicOperNameDiv" style="padding: 10px;border: 1px solid #DEDFE6;width:90%;margin-top: 10px;"></div>
			</div>
		</div>
	</div>
</div>
<c:if test="${isBasic == 'true' }">
    <div class="slideFooter">
        <span class="el-button el-button--primary modifyKpi" onclick="modifyKpiNameCommit()" ><%=rb.getString("QueDing")%></span>
        <span class="el-button" onclick="kpiModifyCancel()"><%=rb.getString("QuXiao")%></span>
   </div>
</c:if>


<script>
var viewKpiNetType = kpiManagePageVue.currentKpiNetType && kpiManagePageVue.currentKpiNetType != '' ? kpiManagePageVue.currentKpiNetType : sysMain.headType;
var searchTextKPIZhiBiao = '';
var kpiId = "";

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
	
	if(viewKpiNetType == 'enb'){
		infoUrl = '${ctx}/pm/indicatormg/getIndicatorInfo.action';
		$('.eNBTypeShow').show();
		$('.eNBModifyFunctionSetShow').show();
		$('.modifyFunctionSetShow').hide();
		//Level 显示
		$('.modifyEnbPlmnLevelShow').show();
	}else if(viewKpiNetType == 'gnb'){
		infoUrl = '${ctx}/gnb/pm/indicatormg/getIndicatorInfo.action';
		$('.eNBTypeShow').hide();
		$('.modifyFunctionSetShow').show();
		$('.eNBModifyFunctionSetShow').hide();
		$('.modifyEnbPlmnLevelShow').show();
	}else{
		infoUrl = '${ctx}/egw/pm/indicatormg/getIndicatorInfo.action';
		$('.eNBTypeShow').hide();
		$('.modifyFunctionSetShow').show();
		$('.eNBModifyFunctionSetShow').hide();
		$('.modifyEnbPlmnLevelShow').show();
	}
	//初始化指标功能集下拉列表
    initFuncSetValue('funcSetValueId_modify');
    //初始化指标单位下拉列表
    initKPIUnit('kpiIdUnitValueId');

    $.post(infoUrl, {"kpiId":kpiId}, function (data) {
		if(data){
			$("#kpiNameValue_view").val(data.kpiName);
			$("#kpiCustomNameValue").val(data.custName);

			if(viewKpiNetType == 'enb'){
				$("#funcSetValueId_modify").combotree("setValue",data.catagoryId);
				$("#funcSetValueId_modify").combotree("setText",data.catagoryName);

				//根据指标类型，判断等级字段是否可修改： 基础指标不可修改等级，需置灰
				if(data.indicatorLevel == 'plmn') {
					$('#kpiTypePlmn_modify').attr('checked',true);
				}else{
					$('#levelTypeEnb_modify').attr('checked',true);
				}
			    initenableSetValue('enableValueId', data.isEnable);
				$("#enableValueId").combobox("setText",data.isEnable);
			}else{
				$("#funcSetValueId_modify").combobox("setText",data.catagoryName);
				$("#funcSetValueId_modify").combobox("setValue",data.catagoryId);
			}
			
			$("#kpiIdUnitValueId").combobox("setText",data.unit);
			
			//初始化统计类型下拉列表
            initStatisticSetValue('StatisticalSetValueId',data["statisType"]); 
			$("#StatisticalSetValueId").combobox("setValue",data.statisType);			
			$("#kpiIdValue").val(data.kpiId);
			$("#explainValue").val(data.definition);
	         
	        $("#calcExpValue").val(data.arithmetic);
	        var calcExpValueShowName = (data.arithmetic);
			var params = data.arithmetic;
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
		   // 设置类型值
		   if(data.indicatorType == 'counter') {
				$('#kpiTypeValue1_modify').attr('checked',true);
				$('#kpi_param_part_modify').hide();
				//$('#kpiNameValue_view').attr('disabled',false);
				if(viewKpiNetType == 'enb'){
					$('.funcSetValueId_modify').combotree('enable');
				}else{
					$('#funcSetValueId_modify').combobox('enable');
				}
				$('#kpiIdUnitValueId,#StatisticalSetValueId,#enableValueId').combobox('enable');

			   // 等级字段可编辑性：只有在修改模式（isBasic='true'）下才允许编辑
			   var isBasicMode = '${isBasic}' === 'true';
			   if(isBasicMode) {
				   // 修改模式：Counter 类型的等级字段可编辑
				   $('#levelTypeEnb_modify,#kpiTypePlmn_modify').prop('disabled', false);
			   } else {
				   // 查看模式：等级字段保持只读
				   $('#levelTypeEnb_modify,#kpiTypePlmn_modify').prop('disabled', true);
			   }
		   }else {
				$('#kpiTypeValue_modify').attr('checked',true);
			   $('#kpi_param_part_modify').show();
			   // KPI 类型的等级字段始终只读
			   $('#levelTypeEnb_modify,#kpiTypePlmn_modify').prop('disabled', true);
		   }
		}
	 }, "json");
	
	/*  阻止冒泡  */
	$('#kpiManagePage .chose').click(function(event){
	    event.stopPropagation()      
	})
	
	if(!(is_super_user || is_build_user)) {
		$('#kpiCustomNameValue').attr('disabled',true);
	}
})

function modifyKpiNameCommit(){
	if(!$(".modifyKpi").hasClass("forbidden")){
		$(".modifyKpi").addClass("forbidden");
		var indicatorType = $('[name=indicatorType]:checked').val();
		var saveUrl = '', 
			params = {
				kpiId: kpiId,
				indicatorType: indicatorType,
				custName: $('#kpiCustomNameValue').val().trim()
			};
		
		if(viewKpiNetType == 'enb'){
			//是否启用
		  	var isEnableValue =  $("#enableValueId").combobox("getValue");
			params.isEnable = isEnableValue;
			//Level 类型：enb / plmn
			var levelType = $('.slideBody [name=modifyLevelType]:checked').val();
			params.indicatorLevel = levelType;

			saveUrl = '${ctx}/pm/indicatormg/updateBaseKpiCustName.action'; // 修改基础指标的下发接口
		}else if(viewKpiNetType == 'gnb'){
			saveUrl = '${ctx}/gnb/pm/indicatormg/updateBaseKpiCustName.action';
		}else{
			saveUrl = '${ctx}/egw/pm/indicatormg/updateBaseKpiCustName.action';
		}

		if(indicatorType == 'counter') {
			//提交请求
			if(viewKpiNetType == 'enb'){
				params.catagoryId = $('#funcSetValueId_modify').combotree('getValue');
				saveUrl = '${ctx}/pm/indicatormg/addOrModifyIndicator.action';	
			}else if(viewKpiNetType == 'gnb'){
				params.catagoryId = $('#funcSetValueId_modify').combobox('getValue');
				saveUrl = '${ctx}/gnb/pm/indicatormg/addOrModifyIndicator.action';
			}else{
				params.catagoryId = $('#funcSetValueId_modify').combobox('getValue');
				saveUrl = '${ctx}/egw/pm/indicatormg/addOrModifyIndicator.action';	
			}

			Object.assign(params, {
				"kpiName": $('#kpiNameValue_view').val().trim(),
				"kpiCustomName": $('#kpiCustomNameValue').val().trim(),
				//"catagoryId": $('#funcSetValueId_modify').combobox('getValue'),
				"unit": $('#kpiIdUnitValueId').combobox('getValue'),
				"statisType": $('#StatisticalSetValueId').combobox('getValue'),
				"definition": $('#explainValue').val().trim(),
				"indicatorType": indicatorType,
				"custName": $('#kpiCustomNameValue').val().trim()
			});
		}
		
		$.post(saveUrl, params, function (data) {
	        if (data["success"]) {
	        	$('#kpiDatagrid').datagrid('reload');
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