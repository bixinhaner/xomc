<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<!-- 窗口，新建、修改指标功能集组 -->
<div id="kpiManaTemplate_basic_body">
	<div id="newTemplateBasicInfoDiv" style="margin-bottom:10px;">
     	<div class="omcPageTitleDiv ">
			<ul class="omcPageTitleContainer titleTabsList">
				<li class="active" logtype="report"><%=rb.getString("JiBenXinXi")%></li>
			</ul>
		</div>
  		<div>			
			<div class="optionDetailsStyle">
				<input name="group_id" id="catagory_group_id" type="hidden" value=""/>
				<input name="catagory" id="catagory_code" type="hidden" value=""/>
				<input name="createTime" id="catagory_createTime_old" type="hidden" value=""/>
				<input name="creatorCode" id="catagory_creatorCode_old" type="hidden" value=""/>
				<input name="old_catagory_name" id="old_catagory_name" type="hidden" value=""/>
				<div class="optionDetailsLeft">					
					<label class="inputTittleCss"><%=rb.getString("ZhiBiaoGongNengJiMingCheng")%>：</label>
					<input id="catagory_name" class="inputDivCss border border-box" oldValue="" maxlength="50"/>
					<label class="inputTipCss errorTipStyle"></label>
				</div>
				<div class="optionDetailsRight" style="margin-left:103px;">					
					<label class="inputTittleCss"><%=rb.getString("ChuangJianZhe")%>：</label>
					<input id="catagory_creatorCode" class="inputDivCss border border-box" disabled="disabled" oldValue=""/>
					<label class="inputTipCss errorTipStyle"></label>
				</div>
			</div>
  			<div class="optionDetailsLeft" style="float:left;">					
				<label class="inputTittleCss"><%=rb.getString("ChuangJianShiJian")%>：</label>
				<input id="catagory_createTime" class="inputDivCss border border-box"  disabled="disabled"/>
				<label class="inputTipCss" ></label>
			</div>  
			<div class="optionDetailsRight">					
					<label class="inputTittleCss"><%=rb.getString("XiangXiMiaoShu")%>：</label>
					<textarea id="catagory_createDetailInfo" class="border border-box" style="width:325px;height:130px;resize:none;"></textarea>
					<label class="inputTipCss errorTipStyle"></label>
			</div>		
		</div>    		
   </div>
</div>
<script type="text/javascript">
$(function(){  
    $("#kpi_Search_Text_Mana").bind("keyup", function (event) {
        if (event.keyCode == 13) {
        	var $ele = $("#kpi_Search_Text_Mana").next();
        	addOrModifyQuery($ele)
        }
    });
})

function closekpiManaTemplate(){
	$("#kpiManaTemplate_basic").animate({right:'-2000px'},500);		
}
</script>
