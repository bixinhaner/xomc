<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
</style>
<div id="kpiManaTemplate_body">
	<div style="margin-bottom:10px;">
  		<div>			
			<div class="optionDetailsStyle">
				<div class="optionDetailsLeft" style="display:block;margin-left:40px;">					
					<label class="inputTittleCss"><%=rb.getString("ZhiBiaoGongNengJiMingCheng")%></label>
					<input id="view_group_name" class="inputDivCss border border-box"  maxlength="50"/>
					<label class="inputTipCss errorTipStyle"></label>
				</div>
				<div class="optionDetailsLeft" style="display:block;margin-left:40px;">					
					<label class="inputTittleCss"><%=rb.getString("XiangXiMiaoShu")%></label>
					<textarea id="view_catagory_createDetailInfo" class="border border-box" style="width:325px;height:130px;resize:none;" maxlength="500"/></textarea>
					<label class="inputTipCss errorTipStyle"></label>
				</div>		
		</div>    		
   </div>
   <div style="margin-left:40px;position:absolute;bottom:30px;">
	   <div class="linkbuttonGroup" style="float:left;">
	        <a href="#" class="linkbutton linkbutton_trend addGroup" onclick="addKpiGroupFun()"><span><%=rb.getString("QueDing")%></span></a>
	        <a href="#" class="linkbutton linkbutton_nowanna" onclick="closeKpiGroupFun()"><span><%=rb.getString("QuXiao")%></span></a>
	   </div>
   </div>
</div>
<script type="text/javascript">
var curEnbOrGnb = sessionStorage.getItem('addGroupEnbOrGnb');

$(function(){  
	$("#view_group_name").blur(function(){
		$(this).val($(this).val().trim());
		if($(this).val().trim().length>0){
			$(this).next().html("");
		}else{
			$(this).next().html("<%=rb.getString("QingShuRuZhiBiaoGongNengJiMingCheng")%>");
		}
	})
})

//保存新建指标功能集
function addKpiGroupFun(){
	var catagoryName = $("#view_group_name").val().trim(), createDetailInfo = $("#view_catagory_createDetailInfo").val(), curSaveUrl = '';

	if(catagoryName.length == 0){
		$("#view_group_name").next().html("<%=rb.getString("QingShuRuZhiBiaoGongNengJiMingCheng")%>");
		$("#view_group_name").focus();
		return;
	}
	var params = {
		"catagoryName": catagoryName,
		"description" : createDetailInfo,
	};

	if(curEnbOrGnb == '0'){
		curSaveUrl = '${ctx}/pm/indicatormg/addIndicatorGroup.action';
	}else if(curEnbOrGnb == '1'){
		curSaveUrl = '${ctx}/gnb/pm/indicatormg/addIndicatorGroup.action';
	}else{
		curSaveUrl = '${ctx}/egw/pm/indicatormg/addIndicatorGroup.action';
	}
	
	if(!$(".addGroup").hasClass("forbidden")){
		$(".addGroup").addClass("forbidden");
		$.post(curSaveUrl, params, function(data) {
			if(data["success"]){
				$("#kpiArithmeticTree").tree("reload");
				showMsg('success_msg','<%=rb.getString("ChengGong")%>');
				closeDefaultWindow();
			} else {
				$(".addGroup").removeClass("forbidden");
				showMsg('error_msg',data["message"]);
				closeDefaultWindow();
			}
		}, "json"); 
	}
}
</script>
