<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>

<style>
	.detailMesDiv div{
		display:inline-block;
	}
	#winViewUpgradeFileInfo label{
		margin-bottom:5px;
		display:block;
	}
	.detailMesDiv {
		display: flex;
		flex-wrap: wrap;
	}
	.detailMesDiv > div {
		flex: 1 1 45%;
	}
	.el-card__body{
		display:flex;
		flex-direction:column;
		flex:1 1 auto;
		height:100%;
		overflow:auto;
	}
	.selectedDiv {
		display:inline-block;
		line-height:20px;
		padding:0 15px;
		margin-right:20px;
		border:1px solid #4D84FF;
		margin-bottom:10px;
	}
	.selectedDiv .el-icon-operation-delete {
		font-size:15px;
		margin-left:20px;
	}
	.coverInput {
		position:absolute;
		left:0;
		top:0;
		width:330px;
		height:24px;
		border:none;
	}
</style>
<div id="winViewUpgradeFileInfo" style="display:flex;flex-direction:column;height:100%;">
		<div class='el-card__header'>
			<span><%=rb.getString("XinXi")%></span>
			<span class='el-icon el-icon-close' onclick='closeViewFileUpradeWindow()'></span>
		</div>
		<div class='el-card__body'>
			<div class='detailMesDiv'  style="background:#fff;padding:20px;">
			    <input name="fileType" type="hidden"/>
			    <div>
					<label for="filePathView" style="width: 120px;"><%=rb.getString("WenJianMing")%></label>
					<input id="filePathView" type="text" class="border border-box file_info" readonly style="width: 350px;"/>
					<span class='operationDiv operation_getFocus fileEditStar'></span>
				</div>
			    <div style=''>
					<label for="versionView" style="width: 120px;"><%=rb.getString("BanBen")%></label>
					<input readonly  id="versionView" type="text" class="border border-box file_info required" maxlength=45 style="width: 350px;height:26px;"/>
					<span class='operationDiv operation_getFocus fileEditStar'></span>
				</div>
				<%-- <div style='margin-top:40px;'>
					<label for="deviceView" style="width: 120px;"><%=rb.getString("SheBeiLeiXing")%></label>
					<input readonly id="deviceView" type="text" class="border border-box file_info required" maxlength=100 style="width: 350px;height:26px;" />
					<span class='operationDiv operation_getFocus fileEditStar'></span>
				</div> --%>
				<div style="vertical-align:top;margin-top:40px;">
	                <label for="recommendView" style="width: 120px;"><%=rb.getString("TuiJian")%></label>
	                <input  id="recommendView" class="border border-box file_info" readonly style="width: 350px;height:26px;"></input>
	                <span class='operationDiv operation_getFocus fileEditStar'></span>
	            </div>
				<div style='margin-top:40px;'>
					<label for="productView" style="width: 120px;"><%=rb.getString("ChanPinLeiXingBiaoZhi")%></label>
					<input readonly id="productView" type="text" class="border border-box file_info required" maxlength=100 style="width: 350px;height:26px;" />
					<span class='operationDiv operation_getFocus fileEditStar'></span>
				</div>
				
				<div style="width:90%;margin-top:40px;">
					<label for="modelNameView" style="width: 120px;"><%=rb.getString("ChanPinXingHao")%></label>
					
					<div class="selectedModuleView" style="margin-top:10px;display:block;width:90%"></div>
				</div>
				<div style="vertical-align:top;margin-top:40px;" id="viewSennYunYingShang">
	                <label for="toWhoView" style="width: 120px;"><%=rb.getString("Title_KaiFangYunYingShang")%></label>
	                <input  id="toWhoView" class="border border-box file_info required" readonly style="width: 350px;height:26px;"></input>
	                <span class='operationDiv operation_getFocus fileEditStar'></span>
	            </div>
				<div style="margin:40px 0px 0px 0px;" id="descViewBox">
					<label for="descView" style="width: 120px; vertical-align: top;"><%=rb.getString("MiaoShu")%></label>
					<textarea readonly id="descView" cols="20" style="outline:none;padding-top:5px;font-size: 12px;width: 766px; height: 377px; resize: none;" rows="5"
						maxlength=500 class="border-box border file_info"></textarea>
				</div>
			</div>
		</div>
</div>
<script>
	$(function(){
		//Cloud版本时，显示可见范围的下拉选项 
		if(isCloudCore == 'true'){
			$("#viewSennYunYingShang").show();
		}else{
			$("#viewSennYunYingShang").hide();
		} 
		$("#modelNameView").combobox({
				url:'${ctx}/cell/CPE/queryModelNames.action',
		    	textField:'model_name',
		    	valueField:'model_name',
				onLoadSuccess:function(){
					
				},
				onSelect:function(){
					$(".errorText_modelName").css("visibility","hidden");
				}
			})
		var combox_data = [
			{
				value:"all",
				label:'<%=rb.getString("ShangYongBanBen")%>'
			},
			{
				value:"none",
				label:'<%=rb.getString("CeShiBanBen")%>',
				selected:true
			},
			{
				value:"beta",
				label:'<%=rb.getString("BetaBanBen")%>'
			}
		];

		$("#toWhoView").combobox({
	    	editable : false,
	        valueField : "value",
	        textField : "label",
	        data : combox_data
	    });
		var recommend_data = [{value:"1",label:"<%=rb.getString("Shi")%>",selected:true},{value:"0",label:"<%=rb.getString("Fou")%>"}];
		  
		$("#recommendView").combobox({
	    	editable : false,
	        valueField : "value",
	        textField : "label",
	        data : recommend_data
	    });
		if(isCpe != 1){
		    $('#winViewUpgradeFileInfo #productView').combobox({
		    	url:'${ctx}/cell/version/getProductType.action?type=all',
		    	textField:'name',
		    	valueField:'value'
		    })
		}else{
			if(operateType == 'ups'){
				
			}else{
				var product_data = [];
				if(writableMap['CODE_CPE_UPGRADE_IMAGE']){
					product_data.push({value:'CPE_VERSION',label:'ODU'});
				}
				if(writableMap['CODE_CPE_UPGRADE_IMAGE']){
					product_data.push({value:'CPE_IDU_VERSION',label:'IDU'});
				}
				$("#productView").combobox({
			    	editable : false,
			        valueField : "value",
			        textField : "label",
			        data : product_data
			    });
			}
		}
	})
</script>