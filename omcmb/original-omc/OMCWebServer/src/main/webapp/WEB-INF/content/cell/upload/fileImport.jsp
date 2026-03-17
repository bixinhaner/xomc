<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>

<style>
	.detailMesDiv div{
		display:inline-block;
	}
	#winUpgradeFileInfo label{
		display:block;
		margin-bottom:5px;
	}
	#winUpgradeFileInfo p{
		font-size:12px;
		color:#CC0000;
		margin-top:5px;
		margin-bottom:7px;
		visibility:hidden;
	}
	#fileUpgradesuccess{
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
<div id="winUpgradeFileInfo" style="display:flex;flex-direction:column;height:100%;">
		<div class='el-card__header'>
			<span><%=rb.getString("WenJianXinXi")%></span>
			<span class='el-icon el-icon-close' onclick='closeImportFileWindow()'></span>
		</div>
		<div class='el-card__body'>
			<div class='detailMesDiv'  style="background:#fff;padding:20px;">
			    <input name="fileType" type="hidden"/>
			    <div class=''>
					<label for="filePath" style="width: 120px;"><%=rb.getString("WenJian")%><%=rb.getString("MaoHao")%></label>
					<input id="filePath" type="text" class="border border-box file_info" readonly="readonly" style="vertical-align:middle;padding-right:27px;width:350px;"/>
					<a class="el-icon el-icon-operation-import" style='vertical-align:middle;display:inline-block;margin:0 2px 0 -29px;' title="<%=rb.getString("XuanZeWenJian")%>" href="javascript: void(0)" onclick="scanClick()" 
			    	style="vertical-align:middle; margin:0 2px 0 -29px;display:inline-block;width:23px;height:24px;background-color:#fff;">
					</a>
					<span class="operationDiv operation_getFocus"></span>
					<p><%=rb.getString("QingXianXuanZeWenJian")%></p>
				</div>
			    <div style=''>
					<label for="version" style="width: 120px;"><%=rb.getString("BanBen")%></label>
					<input onblur='checkVersion()' id="version" type="text" class="border border-box file_info required" maxlength=45 style="width: 350px;height:26px;"/>
					<span class='operationDiv operation_getFocus fileEditStar'></span>
					<p><%=rb.getString("QingShuRuWenJianBanBen")%></p>
				</div>
				<div style="vertical-align:top;margin-top:20px;">
	                <label for="recommend" style="width: 120px;"><%=rb.getString("TuiJian")%></label>
	                <input id="recommend" class="border border-box file_info" style="width: 350px;height:26px;"></input>
	                <span class='operationDiv operation_getFocus fileEditStar'></span>
	                <p><%=rb.getString("ShuRuBiTianXiang")%></p>
	            </div>
				<div style='margin-top:20px;'>
					<label for="product" style="width: 120px;"><%=rb.getString("ChanPinLeiXingBiaoZhi")%></label>
					<input id="product" type="text" class="border border-box file_info required" maxlength=100 style="width: 350px;height:26px;" />
					<span class='operationDiv operation_getFocus fileEditStar'></span>
					<p id='selectType'><%=rb.getString("ShuRuBiTianXiang")%></p>
				</div>
				<div style="width:90%;margin-top:18px">
					<label for="modelName" style="width: 120px;"><%=rb.getString("ChanPinXingHao")%></label>
					<input id="modelName" type="text" class="border border-box file_info required" maxlength=100 style="width: 350px;height:26px;" />
					<span class='el-icon el-icon-plus' onclick="addModule()"></span>
					<div class="selectedModule" style="margin-top:10px;display:block;width:90%"></div>
					<p class="errorText_modelName"><%=rb.getString("QingXuanZeChanPinXingHao")%></p>
				</div>
				<div style="vertical-align:top;margin-top:20px;" id="seenYunYingShang">
	                <label for="toWho" style="width: 120px;"><%=rb.getString("Title_KaiFangYunYingShang")%></label>
	                <input id="toWho" class="border border-box file_info" style="width: 350px;height:26px;"></input>
	                <span class='operationDiv operation_getFocus fileEditStar'></span>
	                <p><%=rb.getString("ShuRuBiTianXiang")%></p>
	            </div>
	            
				<div style="margin-top:20px;" id="descEditBox">
					<label for="desc" style="width: 120px; vertical-align: top;"><%=rb.getString("MiaoShu")%></label>
					<textarea id="desc" cols="20" style="padding-top:5px;font-size: 12px;width: 766px; height: 377px; resize: none;" rows="5"
						maxlength=500 class="border-box border file_info"></textarea>
				</div>
			</div>
		</div>
		<div class="slideFooter" >
			<a id='upgradeSubmit' class="linkbutton linkbutton_trend" onclick='submitUploadForm()'><span><%=rb.getString("QueDing")%></span></a>
			<a class="linkbutton linkbutton_nowanna " onclick="closeImportFileWindow();"><span><%=rb.getString("QuXiao")%></span></a>
		</div>
		<div id='fileUpgradesuccess'><%=rb.getString("XiuGaiWanCheng")%></div>
</div>
<script>
	var deviceTypeArr = [];
	var selText = '',
		selValue = '';
	var midPageType = '${type}';
	$(function(){
		$("#uploadFile_filelib").bind("change", function() {
			$("#winUpgradeFileInfo #filePath").val(this.value);
			 var filePath = $("#uploadFile_filelib").val();
			 var pathSplit = filePath.split(/\\/);
			 var filename = pathSplit[pathSplit.length - 1];
			 if(filename.length > 100){
				 $("#filePath").siblings("p").css("visibility","visible").html("<%=rb.getString("WenJianMingBuNengChaoGuoYaoQiu")%>");
			 }else{
				 if(isCpe != 1){
					 var fileType = $("#uploadFileForm_filelib [name=fileType]").val();
					 if(fileType=='ca'){
						 if(fileFormatMatch(this.value,"patch")){
							 $("#filePath").siblings("p").css("visibility","hidden");
							 $("#winUpgradeFileInfo #version").val(filename.substring(0,filename.lastIndexOf(".")));
							 checkVersion();
						 }else{
							 $("#filePath").siblings("p").css("visibility","visible").html("<%=rb.getString("ZhiZhiChiPATCHWenJian")%>");
						 }
					 }else{
						 if(fileFormatMatch(this.value,"IMG")){
							 $("#filePath").siblings("p").css("visibility","hidden");
							 $("#winUpgradeFileInfo #version").val(filename.substring(0,filename.lastIndexOf(".")));
							 checkVersion();
						 }else{
							 $("#filePath").siblings("p").css("visibility","visible").html("<%=rb.getString("ZhiZhiChiIMGWenJian")%>");
						 }
					 }
				 }else{
					 if(fileFormatMatch(this.value,"tgz,bin")){
						 $("#filePath").siblings("p").css("visibility","hidden");
						 $("#winUpgradeFileInfo #version").val(filename.substring(0,filename.lastIndexOf(".")));
						 checkVersion();
					 }else{
						 $("#filePath").siblings("p").css("visibility","visible").html("<%=rb.getString("ZhIZhiChiTGZBINWenJian")%>");
					 }
				 }
			 }
		});
		$("#modelName").combobox({
				url:'${ctx}/cell/CPE/queryModelNames.action',
		    	textField:'model_name',
		    	valueField:'model_name',
				selectOnNavigation:false,
				multiple:true,
				onSelect: function(record){
					var selDiv = "<div class='selectedDiv'>" + record.model_name + "<span class='el-icon el-icon-operation-delete' onclick='unselectModule(this)'></span>" +"</div>";
					$(".selectedModule").append(selDiv);
					var newText = $("#modelName").combobox("getText");
					selText += newText;
					var newValue = $("#modelName").combobox("getValues");
					selValue = newValue;
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
					$("#modelName").next(".textbox").append("<input id='realInput' class='coverInput' />");
					$("#realInput").keyup(function(ev){
						var nowText = $("#modelName").combobox("getText");
						var addText = $("#realInput").val(),
							text = addText||'';
						
						var ops = $("#modelName").combobox('options');

						$("#modelName").combobox('showPanel');
						ops.keyHandler.query.call($("#modelName")[0],text);
						
						if(selValue) {
							$("#modelName").combobox("setValues",selValue);
						}
					})
				},
				filter:function(q,row){
					
					var opts = $(this).combobox('options');
					var realVal = $("#realInput").val(),
						text = (realVal||'').trim();
					
					return row[opts.textField].indexOf(text) >= 0;
				}
			})
		//Cloud版本时，显示可见范围的下拉选项 
		if(isCloudCore == 'true'){
			$("#seenYunYingShang").show();
		}else{
			$("#seenYunYingShang").hide();
		} 
		
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
		
		$("#toWho").combobox({
	    	editable : false,
	        valueField : "value",
	        textField : "label",
	        data : combox_data
	    });
		var recommend_data = [{value:"1",label:"<%=rb.getString("Shi")%>",selected:true},{value:"0",label:"<%=rb.getString("Fou")%>"}];
		  
		$("#recommend").combobox({
	    	editable : false,
	        valueField : "value",
	        textField : "label",
	        data : recommend_data
	    });
		if(isCpe != 1){
			/* $('#winUpgradeFileInfo #deviceType').combobox({
		    	url:'${ctx}/cell/version/getProductType.action?type=all',
		    	textField:'name',
		    	valueField:'value',
		    	editable:false,
		    	onSelect:function(){
		    		$("#selectDType").css("visibility","hidden");
		    		$('#winUpgradeFileInfo #product').combobox('clear');
		    		$('#winUpgradeFileInfo #product').combobox('reload');
		    	},
		    	loadFilter: function(data){
		    		var arr = [],keys = [];
		    		if(data){
		    			data.map(function(row){
		    				var code = row.device_type;
			    			var item = {name: code, value: code};
			    			if(!keys.includes(code)) {
			    				arr.push(item);
				    			keys.push(code);
			    			}
		    			})
		    		}
		    		return arr;
		    	},
		    	onLoadSuccess: function(data){
		    		var ctn = $(this), has = true;
		    		if(data){
		    			data.map(function(item){
		    				if(item.value == 'Type R' && has){
		    					setTimeout(function(){
			    					ctn.combobox('setValue',item.value);
		    					},100);
		    					has = false;
		    				}
		    			});
		    		}
		    	}
		    });  */
		    $('#winUpgradeFileInfo #product').combobox({
		    	url:'${ctx}/cell/version/getProductType.action?type=all',
		    	textField:'name',
		    	valueField:'value',
		    	editable:false,
		    	onSelect:function(){
		    		$("#selectType").css("visibility","hidden");
		    	},
				onLoadSuccess:function(){
		    		var data = $('#winUpgradeFileInfo #product').combobox("getData");
		    		deviceTypeArr = data;
		    	}
		    	/* loadFilter: function(data){
		    		var orin = data;
		    		try{
			    		var deviceType = $('#winUpgradeFileInfo #deviceType').combobox('getValue')||'Type R';
			    		if(deviceType) orin = orin.filter(function(item){ return item.device_type == deviceType});
			    		return orin;
		    		}catch(e){
		    			return data;
		    		}
		    	} */
		    })
		    
		}else{
			if(operateType == 'enb'){
				var product_data = [];
				if(writableMap['CODE_CPE_UPGRADE_IMAGE']){
					product_data.push({value:'CPE_VERSION',label:'ODU',selected:true});
				}
				if(writableMap['CODE_CPE_UPGRADE_IMAGE']){
					product_data.push({value:'CPE_IDU_VERSION',label:'IDU'});
				}
				$("#product").combobox({
			    	editable : false,
			        valueField : "value",
			        textField : "label",
			        data : product_data
			    });
			}
			//$('#winUpgradeFileInfo #deviceType').parent().remove();
		}
	})
	// 打开窗口，选择文件
	function scanClick() {
		$('#uploadFile_filelib').click();
	}
	// input失去焦点时候 的验证
	function checkVersion(){
		var value = $("#version").val();
		if(value == ""){
			$("#version").siblings("p").html("<%=rb.getString("QingShuRuWenJianBanBen")%>").css("visibility","visible");
			return false;
		}
		if(value.length > 45){
			$("#version").siblings("p").html("<%=rb.getString("FanWei")%>:1-45 <%=rb.getString("ZiFuFuShu")%>").css("visibility","visible");
			return false;
		}
		$("#version").siblings("p").html("").css("visibility","hidden");
		return true;
	}
	if(operateType == 'ups'){
		$("#product").bind("blur",function(e){
			if($("#product").val() == ""){
				$("#selectType").css("visibility","visible");
			}else{
				$("#selectType").css("visibility","hidden");
			}
		})
	}else{
		setTimeout(function(){
			$("#product").combobox('textbox').bind("blur",function(e){
				if($("#product").combobox('getValue') == ""){
					$("#selectType").css("visibility","visible");
				}else{
					$("#selectType").css("visibility","hidden");
				}
			})
		},50);
	}
	
	function addModule(){
		var realVal = $("#realInput").val();
		if (realVal.length > 0){
			var selDiv = "<div class='selectedDiv'>" + realVal + "<span class='el-icon el-icon-operation-delete' onclick='unselectModule(this)'></span>" +"</div>";
			$(".selectedModule").append(selDiv);
			$("#realInput").val('');
			$(".selectedModule").siblings("p").css("visibility","hidden");
		}else{
			
		}
		
	}
	//取消或删除已选择的model 
	function unselectModule(ele){
		var json =$("#modelName").combobox('getData');
		var thisModel = $(ele).parent('.selectedDiv').text();
		$.each(json,function(i){
			if (thisModel == this.model_name){
				$("#modelName").combobox('unselect',this.model_name)
			}
		})
		$(ele).parent('.selectedDiv').remove();
	}
	
	// 确定按钮
	function submitUploadForm(){
		if(!$("#upgradeSubmit").hasClass("forbidden")){
			if($("#filePath").val() == ""){
				$("#filePath").siblings("p").css("visibility","visible").html("<%=rb.getString("QingXianXuanZeWenJian")%>");
			}
			checkVersion();
			if(operateType == 'ups'){
				if($("#product").val() == ""){
					$("#selectType").css("visibility","visible");
				}
			}else{
				if($("#product").combobox('getValue') == ""){
					$("#selectType").css("visibility","visible");
				}
			}
					
			var selectedName = '';
			if ($(".selectedDiv").length > 0){
				$(".selectedDiv").each(function(){
					selectedName = selectedName + $(this).text() +',';
				})
				selectedName = selectedName ? selectedName.substring(0,selectedName.length-1):selectedName;
			}else {
				$(".errorText_modelName").css("visibility","visible");
			}
			
			var isSubmitFlag = true;
			$("#winUpgradeFileInfo p").map(function(index,item){
				if($(item).css("visibility") == "visible"){
					isSubmitFlag = false;
				}
			})
			
			if(isSubmitFlag){
				var files = document.getElementById("uploadFile_filelib").files;
			    var filePath = $("#uploadFile_filelib").val();
				var productDom = $("#winUpgradeFileInfo #product"),
					productValue = operateType=='ups'? productDom.val():productDom.combobox('getValue');
				if(isCpe == 1){//cpe
					deviceValue = ''
				}else{//非cpe
					deviceTypeArr.map(function(item,index){
						if(item.value == productValue){
							deviceValue = item.device_type;
						}
					})
				}
				/* var deviceDom = $("#winUpgradeFileInfo #deviceType"),
					deviceValue = isCpe==1? '':deviceDom.combobox('getValue'); */
			    $("#winUpgradeFileInfo #product").val(productValue);
			    var pathSplit = filePath.split(/\\/);
			    var filename = pathSplit[pathSplit.length - 1];
			    // 向form中赋值
			    $("#uploadFileForm_filelib [name=fileSize]").val(files[0].size);
			    $("#uploadFileForm_filelib [name=newFileName]").val(filename);
			    if($("#winUpgradeFileInfo #product").size()>0){
			        $("#uploadFileForm_filelib [name=product]").val(productValue);
			    }else $("#uploadFileForm_filelib [name=product]").val("");
			    if(isCpe == 1){
			    	$("#uploadFileForm_filelib [name=deviceType]").val("");
			    }else{
			    	$("#uploadFileForm_filelib [name=deviceType]").val(deviceValue);
			    }
			    /* if(deviceDom.size()>0){
			        $("#uploadFileForm_filelib [name=deviceType]").val(deviceValue);
			    }else $("#uploadFileForm_filelib [name=deviceType]").val("");*/
			    
			    $("#uploadFileForm_filelib [name=version]").val($("#winUpgradeFileInfo #version").val());
				$("#uploadFileForm_filelib [name=model_name]").val(selectedName);
				
			    if($("#winUpgradeFileInfo #seenYunYingShang").is(":visible")) {
			    	$("#uploadFileForm_filelib [name=to_who]").val($("#winUpgradeFileInfo #toWho").combobox("getValue"));
			    }else $("#uploadFileForm_filelib [name=to_who]").val("All");
			    $("#uploadFileForm_filelib [name=recommend]").val($("#winUpgradeFileInfo #recommend").combobox("getValue"));
			    $("#uploadFileForm_filelib [name=desc]").val($("#winUpgradeFileInfo #desc").val().replace(/\n/g, " "));
			    $('#progressUploadFile').progressbar('setValue', 0);// 将进度条进度置为0
			    $("#winUploadPro").window("open");// 打开进度条窗口
			    intervalGetProgress = window.setInterval(function(){
		    		getUploadProgress();
					var dom = $("#progressUploadFile");
					if(dom.length == 0) clearInterval(intervalGetProgress);
			    }, 1000);// 定时读取进度
			    var grid;
			    if(isCpe && operateType == 'enb'){
			    	if(productValue == 'CPE_VERSION'){
			    		$("#uploadFileForm_filelib [name=fileType]").val('upgradecpe')
			    	}else{
			    		$("#uploadFileForm_filelib [name=fileType]").val('upgradeiducpe')
			    	}
			    }
			    var fileType = $("#uploadFileForm_filelib [name=fileType]").val();
			 
			    if (fileType == "upgrade") {
			        grid = $("#fileinfo_upgrade");
			    } else if (fileType == "bios") {
			        grid = $("#fileinfo_bios");
			    } else if (fileType == "ca") {
			        grid = $("#fileinfo_ca");
			    } else if (fileType == "fpga") {
			        grid = $("#fileinfo_fpga");
			    } else if (fileType == "upgradecpe") {
			    	grid = $("#fileinfo_upgradecpe");
			    } else if (fileType == "upgradeiducpe") {
			    	grid = $("#fileinfo_upgradecpe");
			    }else if (fileType == "ups") {
			    	grid = $("#fileinfo_upgradeups");
			    }
			 
			    uploadWithProgress({
			    	url: "${ctx}/cell/version/uploadVersionFile.action",
			    	form: document.querySelector("#uploadFileForm_filelib"),
			    	progress: function(ev){
			    		if(ev.lengthComputable || ev.event.lengthComputable) {
				    		var total = ev.total,
				    			loaded = ev.loaded,
				    			percent = 100*loaded/total;
				    		$('#progressUploadFile').progressbar('setValue', percent.toFixed(2));
			    		}
			    	},
			    	success: function(data){
			        	if(typeof data == 'string') data = eval('('+data+')');
			    		// 清除定时器
			            window.clearInterval(intervalGetProgress);
			            $("#winUploadPro").window("close");// 关闭进度条窗口
			    		if (data["MD5"]) {
			            	$("#upgradeSubmit").addClass("forbidden");
			            	document.getElementById("uploadFile_filelib").value = "";// 置空文件组件
			            	$.messager.alert(TiShi,"<%=rb.getString("ShangChuanChengGong")%><%=rb.getString("DouHao")%><%=rb.getString("WenJianMD5Zhi")%><%=rb.getString("MaoHao")%>"+data["MD5"]);
			            	closeImportFileWindow();
			            	grid.datagrid("reload");
			            } else {
			            	//$.messager.alert(TiShi, data["message"]);
			            	showMsg('prompt_msg',data["message"]);
			            }
			    	}
			    });
			}
		}
	}
	
</script>