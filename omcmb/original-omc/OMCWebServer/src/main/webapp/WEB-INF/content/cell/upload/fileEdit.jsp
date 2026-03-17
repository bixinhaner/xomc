<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>

<style>
	.detailMesDiv div{
		display:inline-block;
	}
	#winEditUpgradeFileInfo label{
		margin-bottom:5px;
		display:block;
	}
	#winEditUpgradeFileInfo p{
		font-size:12px;
		color:#CC0000;
		margin-top:5px;
		margin-bottom:7px;
		visibility:hidden;
	}
	#editFileUpgradesuccess{
		background:#E9FBFF url("${ctx}/images/success.png") no-repeat 8px center;
		height:38px;
		float:left;
		margin-left:30px;
		color:#508D9B;
		font-size:15px;
		font-weight:bold;
		padding:0 10px 0 10px;
		line-height:38px;
		text-indent:25px;
		display:none;
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
	.combobox-item-selected {
		background:#F2F6FF;
	}
	.combobox-item-selected::after {
		content:'\e734';
		float:right;
		margin-right:15px;
		font-family:'el-icon';
		font-size:14ppx;
		color:#4D84FF;
	}
	.combobox-item-hover {
		background:#F2F6FF;
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
<div id="winEditUpgradeFileInfo" style="display:flex;flex-direction:column;height:100%;">
		<div class='el-card__header'>
			<span><%=rb.getString("WenJianXinXi")%></span>
			<span class='el-icon el-icon-close' onclick='closeFileUpgradeWindow()'></span>
		</div>
		<div class='el-card__body'>
			<div class='detailMesDiv'  style="background:#fff;padding:20px;">
		    		<input name="fileType" type="hidden"/>
			    	<div>
					<label for="filePathEdit" style="width: 120px;"><%=rb.getString("WenJianMing")%></label>
					<input id="filePathEdit" type="text" class="border border-box file_info" readonly style="width: 350px;"/>
					<span class='operationDiv operation_getFocus fileEditStar'></span>
					<p style='visibility:hidden'><%=rb.getString("ShuRuBiTianXiang")%></p>
				</div>
			    	<div style=''>
					<label for="versionEdit" style="width: 120px;"><%=rb.getString("BanBen")%></label>
					<input onblur='checkVersion()' id="versionEdit" type="text" class="border border-box file_info required" maxlength=45 style="width: 350px;height:26px;"/>
					<span class='operationDiv operation_getFocus fileEditStar'></span>
					<p><%=rb.getString("QingShuRuWenJianBanBen")%></p>
				</div>
				<div style="vertical-align:top;margin-top:20px;">
	                		<label for="recommend" style="width: 120px;"><%=rb.getString("TuiJian")%></label>
	                		<input id="recommendEdit" class="border border-box file_info" style="width: 350px;height:26px;"></input>
	               	 		<span class='operationDiv operation_getFocus fileEditStar'></span>
	                		<p><%=rb.getString("ShuRuBiTianXiang")%></p>
	            		</div>
				<div style='margin-top:20px;'>
					<label for="productEdit" style="width: 120px;"><%=rb.getString("ChanPinLeiXingBiaoZhi")%></label>
					<input id="productEdit" type="text" class="border border-box file_info required" maxlength=100 style="width: 350px;height:26px;" />
					<span class='operationDiv operation_getFocus fileEditStar'></span>
					<p id='selectType'><%=rb.getString("ShuRuBiTianXiang")%></p>
				</div>
				<div style="width:90%;margin-top:18px">
					<label for="modelName" style="width: 120px;"><%=rb.getString("ChanPinXingHao")%></label>
					<input id="modelNameEdit" type="text" class="border border-box file_info required" maxlength=100 style="width: 350px;height:26px;" />
					<span class='el-icon el-icon-plus' onclick="addModule()"></span>
					<div class="selectedModuleEdit" style="margin-top:10px;display:block;width:90%"></div>
					<p class="errorText_modelName"><%=rb.getString("QingXuanZeChanPinXingHao")%></p>
				</div>
				<div style="vertical-align:top;margin-top:20px;" id="editSennYunYingShang">
	                <label for="toWhoEdit" style="width: 120px;"><%=rb.getString("Title_KaiFangYunYingShang")%></label>
	                <input id="toWhoEdit" class="border border-box file_info" style="width: 350px;height:26px;"></input>
	                <span class='operationDiv operation_getFocus fileEditStar'></span>
	                <p><%=rb.getString("QingXuanZeChanPinLeiXing")%></p>
	            </div>
				<div style="margin-top:20px;" id="descEditBox">
					<label for="descEdit" style="width: 120px; vertical-align: top;"><%=rb.getString("MiaoShu")%></label>
					<textarea id="descEdit" cols="20" style="padding-top:5px;font-size: 12px;width: 766px; height: 377px; resize: none;" rows="5"
						maxlength=500 class="border-box border file_info"></textarea>
				</div>
			</div>
		</div>
		<div class='el-card__footer'>
			<div class="windowButtonGroup" style="float:left;">
				<a class="linkbutton linkbutton_trend" id="submitUploadEdit"><span><%=rb.getString("QueDing")%></span></a>
				<a class="linkbutton linkbutton_nowanna " onclick="closeFileUpgradeWindow();"><span><%=rb.getString("QuXiao")%></span></a>
			</div>
			<div id='editFileUpgradesuccess'><%=rb.getString("XiuGaiWanCheng")%></div>
		</div>

</div>
<script>
	$(function(){
		//Cloud版本时，显示可见范围的下拉选项 
		if(isCloudCore == 'true'){
			$("#editSennYunYingShang").show();
		}else{
			$("#editSennYunYingShang").hide();
		} 
		$("#modelNameEdit").combobox({
				url:'${ctx}/cell/CPE/queryModelNames.action',
		    	textField:'model_name',
		    	valueField:'model_name',
		    	selectOnNavigation:false,
				multiple:true,
				onSelect: function(record){
					var selDiv = "<div class='selectedDiv'>" + record.model_name + "<span class='el-icon el-icon-operation-delete' onclick='unselectModule(this)'></span>" +"</div>";
					$(".selectedModuleEdit").append(selDiv);
					var newText = $("#modelNameEdit").combobox("getText");
					selText += newText;
					var newValue = $("#modelNameEdit").combobox("getValues");
					$(".errorText_modelName").css("visibility","hidden");
					
				},
				onUnselect: function(record){
					$(".selectedDiv").each(function(){
						if ($(this).text() == record.model_name){
							$(this).remove()
						}
					})
					
				},
				onLoadSuccess:function(){
					$("#modelNameEdit").next(".textbox").append("<input id='realInput' class='coverInput' />");
					
					$("#realInput").keyup(function(ev){
						
						var nowText = $("#modelNameEdit").combobox("getText");
						var addText = $("#realInput").val();
						$("#modelNameEdit").combobox("setText",selText+','+addText);
						
						$("#modelNameEdit").next().find('input.validatebox-text').trigger('query');
						
					})
				},
				filter:function(q,row){
					
					var opts = $(this).combobox('options');
					var realVal = $("#realInput").val();
					return row[opts.textField].indexOf(selText+','+realVal) == 0;
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

		$("#toWhoEdit").combobox({
	    	editable : false,
	        valueField : "value",
	        textField : "label",
	        data : combox_data
	    });
		var recommend_data = [{value:"1",label:"<%=rb.getString("Shi")%>",selected:true},{value:"0",label:"<%=rb.getString("Fou")%>"}];
		  
		$("#recommendEdit").combobox({
	    	editable : false,
	        valueField : "value",
	        textField : "label",
	        data : recommend_data
	    });
		if(isCpe != 1){
		    $('#winEditUpgradeFileInfo #productEdit').combobox({
		    	url:'${ctx}/cell/version/getProductType.action?type=all',
		    	editable:false,
		    	textField:'name',
		    	valueField:'value',
		    	onSelect:function(){
		    		$("#selectType").css("visibility","hidden");
		    	},
		    	onLoadSuccess:function(){
		    		var data = $('#winEditUpgradeFileInfo #productEdit').combobox("getData");
		    		deviceTypeEditArr = data;
		    	}
		    	/* loadFilter: function(data){
		    		var orin = data;
		    		try{
			    		var deviceType = $('#winEditUpgradeFileInfo #deviceEdit').combobox('getValue');
			    		if(deviceType) orin = orin.filter(function(item){ return item.device_type == deviceType});
			    		return orin;
		    		}catch(e){
		    			return data;
		    		}
		    	} */
		    });
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
				$("#productEdit").combobox({
			    	editable : false,
			        valueField : "value",
			        textField : "label",
			        data : product_data
			    });
			}
		}
	})
	
	
	function addModule(){
		var realVal = $("#realInput").val();
		if (realVal.length > 0){
			var selDiv = "<div class='selectedDiv'>" + realVal + "<span class='el-icon el-icon-operation-delete' onclick='unselectModule(this)'></span>" +"</div>";
			$(".selectedModuleEdit").append(selDiv);
			$("#realInput").val('');
			$(".selectedModuleEdit").siblings("p").css("visibility","hidden");
		}else{
			
		}
		
	}
	
	//取消或删除已选择的model
	function unselectModule(ele){
		var json =$("#modelNameEdit").combobox('getData');
		var thisModel = $(ele).parent('.selectedDiv').text();
		$.each(json,function(i){
			if (thisModel == this.model_name){
				$("#moduleName").combobox('unselect',this.model_name)
			}
		})
		$(ele).parent('.selectedDiv').remove();
	}
	
	function checkVersion(){
		if($("#versionEdit").val() == ""){
			$("#versionEdit").siblings("p").css("visibility","visible");
			return false;
		}else{
			$("#versionEdit").siblings("p").css("visibility","hidden");
			return true;
		}
	}
	if(operateType == 'ups'){
		$("#productEdit").bind("blur",function(e){
			if($("#productEdit").val() == ""){
				$("#selectType").css("visibility","visible");
			}else{
				$("#selectType").css("visibility","hidden");
			}
		})
	}else{
		$("#productEdit").combobox('textbox').bind("blur",function(e){
			if($("#productEdit").combobox('getValue') == ""){
				$("#selectType").css("visibility","visible");
			}else{
				$("#selectType").css("visibility","hidden");
			}
		})
	}
	
</script>