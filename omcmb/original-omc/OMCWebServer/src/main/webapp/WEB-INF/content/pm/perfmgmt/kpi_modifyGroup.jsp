<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<!-- 窗口，新建、修改指标功能集组 -->
<div id="kpiManaTemplate_body">
	<div style="margin-bottom:10px;">
  		<div>			
			<div class="optionDetailsStyle">
				<input id="old_modify_group_name" type="hidden" value=""/>
				<input id="old_modify_group_detailInfo" type="hidden" value=""/>
				<div class="optionDetailsLeft" style="display:block;margin-left:40px;">					
					<label class="inputTittleCss"><%=rb.getString("ZhiBiaoGongNengJiMingCheng")%>：</label>
					<input id="modify_group_name" class="inputDivCss border border-box" oldValue="" maxlength="50"/>
					<div class="operationDiv status_star"></div>
					<label class="inputTipCss errorTipStyle"></label>
				</div>
				<div class="optionDetailsLeft" style="display:block;margin-left:40px;">					
					<label class="inputTittleCss"><%=rb.getString("XiangXiMiaoShu")%>：</label>
					<textarea id="modify_group_detailInfo" class="border border-box" style="width:325px;height:130px;resize:none;" maxlength="500"></textarea>
					<label class="inputTipCss errorTipStyle"></label>
				</div>		
		</div>    		
   </div>
   <div style="margin-left:40px;position:absolute;bottom:30px;">
	   <div class="linkbuttonGroup" style="float:left;">
	        <a href="#" class="linkbutton linkbutton_trend modifyGroup" onclick="modifyKpiGroupFun()"><span><%=rb.getString("QueDing")%></span></a>
        	<a href="#" class="linkbutton linkbutton_nowanna" onclick="closeKpiGroupFun()"><span><%=rb.getString("QuXiao")%></span></a>
   	   </div>
   </div>
</div>
<script type="text/javascript">
var curEnbGnbOrEgw = sessionStorage.getItem('editGroupEnbOrGnb');

$(function(){  
	setTimeout(function(){
		var curInfoUrl = '', node=$('#kpiArithmeticTree').tree("getSelected");

		if(curEnbGnbOrEgw == '0'){
			curInfoUrl = '${ctx}/pm/indicatormg/getIndicatorGroupInfo.action';
		}else if(curEnbGnbOrEgw == '1'){
			curInfoUrl = '${ctx}/gnb/pm/indicatormg/getIndicatorGroupInfo.action';
		}else{
			curInfoUrl = '${ctx}/egw/pm/indicatormg/getIndicatorGroupInfo.action';
		}
		
	   	$.post(curInfoUrl, {timeZone:timeZone,catagoryId:node.id}, function (data) {
	    	if(data){
	 			$("#modify_group_name").val(data.catagoryName);
	 			$("#old_modify_group_name").val(data.catagoryName);
	 			$("#modify_group_detailInfo").val(data.description);
	 			$("#old_modify_group_detailInfo").val(data.description);
	    	}
		}, "json");
	},300);
	
	$("#modify_group_name").blur(function(){
		if($(this).val().trim().length>0){
			$(this).next().next().html("");
		}else{
			$(this).next().next().html("<%=rb.getString("QingShuRuZhiBiaoGongNengJiMingCheng")%>");
		}
	})
})

//添加修改指标功能集
function modifyKpiGroupFun(){
	var node=$('#kpiArithmeticTree').tree("getSelected");
	var catagoryId = node.id;
	var catagoryName = $("#modify_group_name").val().trim();
	var oldCatagoryName = $("#old_modify_group_name").val();
	var modifyDetailInfo = $("#modify_group_detailInfo").val();
	var oldModifyDetailInfo = $("#old_modify_group_detailInfo").val();
	
	if(catagoryName.length == 0){
		$("#modify_group_name").next().next().html("<%=rb.getString("QingShuRuZhiBiaoGongNengJiMingCheng")%>");
		$("#modify_group_name").focus();
		return;
	}
	
	var curSaveUrl = '',
		params = {
		"catagoryId": catagoryId,
		"catagoryName": catagoryName,
		"description" : modifyDetailInfo,
	};
	if(!$(".modifyGroup").hasClass("forbidden")){
		if(catagoryName == oldCatagoryName && modifyDetailInfo == oldModifyDetailInfo ){
			showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>')
			return;
		}
		$(".modifyGroup").addClass("forbidden");
	
		if(curEnbGnbOrEgw == '0'){
			curSaveUrl = '${ctx}/pm/indicatormg/modifyIndicatorGroup.action';
		}else if(curEnbGnbOrEgw == '1'){
			curSaveUrl = '${ctx}/gnb/pm/indicatormg/modifyIndicatorGroup.action';
		}else{
			curSaveUrl = '${ctx}/egw/pm/indicatormg/modifyIndicatorGroup.action';
		}
		
		$.post(curSaveUrl, params, function(data) {
			if (data["success"]) {
				showMsg('success_msg','<%=rb.getString("ChengGong")%>');
				$("#kpiArithmeticTree").tree("reload");
				closeDefaultWindow();
				$('#modifyKpiGroupSuccess').fadeIn(300,function(){
					setTimeout("closeKpiGroupFun()",1500);
				})
			} else {
				$(".modifyGroup").removeClass("forbidden");
				showMsg('error_msg',data["message"]);
			}
		}, "json"); 
	}
}
</script>
