<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<div id="kpiManaModify_body" >
	<div class="slideCont">
		<div class="splitGroup">
			<div class="splitGroup_title"><%=rb.getString("JiBenXinXi")%></div>
	  		<div class="splitGroup_body form-group-inline">				
				<div>
					<label class="inputTittleCss"><%=rb.getString("ZhiBiaoMingCheng")%></label>
					<input id="kpiNameValue" class="inputDivCss border border-box" oldValue="" maxLength="200" onkeyup="this.value=this.value.replace(/^ +| $/g,'')"/>
					<div class="operationDiv status_star"></div>
					<label class="inputTipCss errorTipStyle"></label>
				</div> 
				<div>					
					<label class="inputTittleCss"><%=rb.getString("SuoShuGongNengJi")%></label>
					<select id="funcSetValueId" name="funcSetValueName" class="easyui-combobox inputDivCss border border-box" style="height:26px;" oldValue=""></select>
					<div class="operationDiv status_star"></div>
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
	  				<div class="infoDivTitle"><%=rb.getString("ShuoMing")%></div>
	  				<textarea id="explainValue" class="InfoDivContent" maxLength="2000" onkeyup="this.value=this.value.replace(/^ +| $/g,'')"
	  					style="display:block;height:100px;width:765px;padding:15px 0px 10px 15px;overflow-y:auto;resize:none;"></textarea>
					<label class="inputTipCss errorTipStyle"></label>
  				</div>
  			</div>
		</div>
		<div class="splitGroup">
			<div class="splitGroup_title"><%=rb.getString("JiSuanGongShi")%></div>
	  		<div class="splitGroup_body form-group-inline">	
				<div style="margin-left:6%;margin-top:30px; position: relative;">
					<textarea id="calcExpValue" style="display:none;"></textarea>
					<div id="calcExpValueShow" class="InfoDivContent" style="display:block;height:100px;width:765px;padding:15px 0px 10px 15px;overflow-y:auto;" readonly="readonly"></div>
					<div tabindex="0" class="ivu-tag" style="display:none" id="hiddenTag">
					    <span class="ivu-tag-text" contenteditable="true"> </span>
					    <i class="ivu-icon ivu-icon-ios-close-empty titleIcon_close"></i>
					</div>
				</div>
	            <div style="margin-left: 6%;margin-top:10px;">
		            <div class="windowButtonGroup" style="float:initial;">					            
		                <a class="linkbutton linkbutton_nowanna" onclick="addSign('+')"><span>+</span></a>
		                <a class="linkbutton linkbutton_nowanna" onclick="addSign('-')"><span>-</span></a>
		                <a class="linkbutton linkbutton_nowanna" onclick="addSign('*')"><span>*</span></a>
		                <a class="linkbutton linkbutton_nowanna" onclick="addSign('/')"><span>/</span></a>
		                <a class="linkbutton linkbutton_nowanna" onclick="addSign('(')"><span>(</span></a>
		                <a class="linkbutton linkbutton_nowanna" onclick="addSign(')')"><span>)</span></a>
		                <a class="linkbutton linkbutton_nowanna" onclick="addSign('num')"><span>0-9</span></a>
		                <a class="linkbutton linkbutton_nowanna" onclick="addSign('clear')"><span><%=rb.getString("QingChu")%></span></a>
		            </div>
	            </div>
	            <div style="margin-left: 6%;margin-top:10px;">
	            	<label id="calcExpValueShowTitle" class="inputTipCss errorTipStyle" ></label>
	            </div>
            </div>
            <div class="queryGroup" style="padding-left: 55px;">
				<input id="kpi_search_text" style="width:300px;" placeholder="<%=rb.getString("ZhiBiaoMingChengZhiBiaoJi")%>" />
				<b onclick="reloadKpiTree()"></b>
			</div>
            <div id="kpiAlgorithmicDiv" style="padding:0px 20px;height:430px;">
		        <div class="leftCol" style="margin-left:35px;">
					<div class="infoDivTitle"><%=rb.getString("ZhiBiaoGongNengJi")%></div>
		        	<div class="InfoDivContent" style="overflow-x:hidden" id="kpiSetTree"></div>
		        </div>
		        <div class="rightCol" style="width:570px" >
		        	<div class="infoDivTitle"><%=rb.getString("XingNengZhiBiao")%></div>
					<div class="InfoDivContent">
						<table id="kpiListDatagrid"></table>
					</div>
		        </div>
			</div>
		</div>
   </div>
</div>
<div class="slideFooter">
     <span class="el-button el-button--primary addKpi" onclick="addKPIArithCommit()" ><%=rb.getString("QueDing")%></span>
     <span class="el-button" onclick="kpiModifyCancel()"><%=rb.getString("QuXiao")%></span>
</div>


<script>
var searchTextKPIZhiBiao = '';

$(function(){
	//初始化指标功能集下拉列表
    initFuncSetValue('funcSetValueId');
    //初始化指标单位下拉列表
    initKPIUnit('kpiIdUnitValueId');
    //初始化指标算法选择树
    loadFuncSet();
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
        	reloadKpiTree();
        }
    });
	$("#kpiNameValue").blur(function(){
		if($(this).val().length>0){
			$(this).next().next().html("");
		}else{
			$(this).next().next().html("<%=rb.getString("ZhiBiaoMingChengWeiKong")%>");
		}
	}) 
	$("#funcSetValueId").combobox({onChange:function(){
		$("#funcSetValueIdTittle").html("");
	}})
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
	
    //指标功能集 
    var funcSetValue =  $("#funcSetValueId").combobox("getValue");
  	//指标单位  
    var kpiIdUnitValue =  $("#kpiIdUnitValueId").combobox("getValue");
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
	//指标名称 
    var kpiNameValue = $("#kpiNameValue").val();
    if (kpiNameValue.length === 0) {
    	$("#kpiNameValue").next().next().html("<%=rb.getString("ZhiBiaoMingChengWeiKong")%>");
    	$("#kpiNameValue").focus();
		isBlank=false;
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
    var params = {
    		"kpiName" : kpiNameValue,
    		"catagoryId": funcSetValue,
    		"unit":kpiIdUnitValue,
    		"definition": explainValue,
    		"arithmetic": calcExpValue
    		//"threadData":threadFormData
        };
    //提交请求
   if(!$(".addKpi").hasClass("forbidden")){
		$(".addKpi").addClass("forbidden");
		$.post("${ctx}/pm/indicatormg/addOrModifyIndicator.action", params, function (data) {
	        if (data["success"]) {
	        	try{itemArr = new Array();}catch(e){}
	        	$("#kpiArithmeticTree").tree("reload");
	            $('#addKpiSuccess').fadeIn(300,function(){
					setTimeout("kpiModifyCancel()",1500);
				})
	        } else {
				$(".addKpi").removeClass("forbidden");
	        	showMsg('error_msg',data.message);
	        }
	    }, "json");
	}
}
</script>