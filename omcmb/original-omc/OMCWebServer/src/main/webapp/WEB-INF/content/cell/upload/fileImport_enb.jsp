<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#fileImportEnbDiv .el-form-item__label{
	line-height:26px;
	width:100px;
	text-align:left;
	font-size:12px;
}
#fileImportEnbDiv .el-form-item__error{
	margin-left:100px;
}
</style>

<div id='fileImportEnbDiv'>
	<el-form ref="enbFileImportForm" :model="importForm" :rules="importRules" style="margin-left:10px;margin-top:20px;" :hide-required-asterisk=true>
		<el-form-item prop="product" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" style='display:inline-block;width:315px;' v-if="showProduct" key="product">
			<el-select v-model='importForm.product'>	
				<el-option v-for='item in productOptions' :label="item.name" :value="item.value"></el-option>
			</el-select>
		</el-form-item>
		<el-form-item label="<%=rb.getString("WenJianMing")%>" prop="newFileName">
			<el-input :disabled="true" v-model="importForm.newFileName">
				<i @click="importFile" slot='suffix' style='display:inline-block;width:28px;height:28px;margin:4px -9px 0 0;' class='el-icon el-icon-operation-import'></i>
			</el-input>
			<span style='color:#999;margin-left:10px;'>{{fileTypeTip}}</span>
		</el-form-item>
		<el-form-item label="<%=rb.getString("BanBen")%>" prop="version">
			<el-input v-model='importForm.version'></el-input>
		</el-form-item>
		<el-form-item label='<%=rb.getString("Title_KaiFangYunYingShang")%>' prop='to_who' v-if="showToWho">
			<el-select v-model='importForm.to_who'>
				<el-option label='<%=rb.getString("ShangYongBanBen")%>' value='all'></el-option>
				<el-option label='<%=rb.getString("CeShiBanBen")%>' value='none'></el-option>
				<el-option label="<%=rb.getString("BetaBanBen")%>" value="beta"></el-option>
			</el-select>
		</el-form-item>
		<el-form-item label='<%=rb.getString("TuiJian")%>' prop='recommend'>
			<el-select v-model='importForm.recommend'>
				<el-option label='<%=rb.getString("Shi")%>' value='1'></el-option>
				<el-option label='<%=rb.getString("Fou")%>' value='0'></el-option>
			</el-select>
		</el-form-item>
		<el-form-item label="<%=rb.getString("MiaoShu")%>" prop='desc'>
			<el-input v-model='importForm.desc' type='textarea' :rows='2' style='width:260px;'></el-input>
		</el-form-item>
	</el-form>
</div>
<%-- 表单-上传基站列表文件 --%>
<form enctype="multipart/form-data" method="post" id="uploadForm_enbFileImport" style="display: none;">
    <input name="uploadFile" type="file" id="uploadFileImportEnb">
    <input name="newFileName" id="newFileName" value="" hidden="true">
    <input name="fileSize" value="" hidden="true">
    <input name="fileType" id="fileType" value="" type="hidden"/>
    <input name="deviceType" value="" type="hidden"/>
    <input name="product" value="" type="hidden"/>
    <input name="version" value="" type="hidden"/>
    <input name="to_who" value="" type="hidden"/>
    <input name="desc" value="" type="hidden"/>
    <input name="md5" value="" type="hidden"/>
    <input name="recommend" value="" type="hidden"/>
</form>
<script>
	var enbFileImportVue = new Vue({
		el:'#fileImportEnbDiv',
		data(){
			var vm = this;
			var validateFilePath = (rule,value,callback) => {
				if(enbFileVue.file_type == 'upgrade' && vm.productLabel == 'DXDF'){
					var reg = vm.difFileObj[enbFileVue.file_type].fileFmtYD;
					var message = vm.difFileObj[enbFileVue.file_type].fileErrorMsgYD;
				}else if((vm.productLabel == 'CR-B4860/EU' || vm.productLabel == 'CR-B4860/RU') && enbFileVue.file_type == 'upgrade'){
					var reg = vm.difFileObj[enbFileVue.file_type].fileFmt4860;
					var message = vm.difFileObj[enbFileVue.file_type].fileErrorMsg4860;
				}else{
					var reg = vm.difFileObj[enbFileVue.file_type].fileFmt;
					var message = vm.difFileObj[enbFileVue.file_type].fileErrorMsg;
				}
				if(value == ""){
					callback(new Error("<%=rb.getString("QingXianXuanZeWenJian")%>"))
				}else if(!vm.fileFormatMatch(value,reg)){
					callback(new Error(message))
				}else{
					var pathSplit = value.split(/\\/),
						filename = pathSplit[pathSplit.length - 1];
					
					if(filename.length>100) {
						callback("<%=rb.getString("WenJianMingBuNengChaoGuoYaoQiu")%>");
					}else {
						callback();
					}
				}
			},
			versionValidate = function(rule,value,callback) {
				if(value) {
					if(value.length>45) {
						callback('<%=rb.getString("FanWei")%>:1-45 <%=rb.getString("ZiFuFuShu")%>');
					}else {
						var nxpObj = {
								"CR-B4860/EU" : 'eu',
								"CR-B4860/BU" : 'bu',
								"CR-B4860/RU" : 'ru'
						}
						var fileType = vm.importForm.product.includes("CR-B4860") ? vm.importForm.product : enbFileVue.file_type;
						axios.get("${ctx}/cell/version/verifyUVExist.action",{
							params:{
								fileType : fileType,
								version : value
							}
						}).then(function(response){
							var data = response.data;
							if(data){
								callback("<%=rb.getString("Msg_ShengJiWenJianBanBenYiJingCunZai")%>");
							}else{
								callback();
							}
						})
					}
				}else {
					callback("<%=rb.getString("QingShuRuWenJianBanBen")%>");
				}
			};
			return{
				importForm:{
					product:'',
					newFileName:'',
					version:'',
					recommend:'1',
					desc:'',
					to_who:'all'
				},
				importRules:{
					newFileName:[
						{validator: validateFilePath}
					],
					version:[
						{validator: versionValidate,trigger:'blur'}
					],
					product:[
						{required:true,message:'<%=rb.getString("ShuRuBiTianXiang")%>',trigger:'change'}
					],
					recommend:[
						{required:true,message:'<%=rb.getString("ShuRuBiTianXiang")%>',trigger:'change'}
					]
				},
				productOptions:[],
				showToWho:false,
				difFileObj:{
					upgrade:{
						fileTip:'<%=rb.getString("ZhiChiIMGGeShi")%>',
						fileTipYD:'<%=rb.getString("ShengJiZhiChiGeShi")%>',
						fileTip4860:'<%=rb.getString("ZhiZhiChiIMGHeEXTGeShi")%>',
						fileFmt:'IMG',
						fileFmtYD:'tar.gz',
						fileFmt4860:'IMG,EXT',
						fileErrorMsg:'<%=rb.getString("ZhiZhiChiIMGWenJian")%>',
						fileErrorMsgYD:'<%=rb.getString("ShengJiZhiZhiChiWenJian")%>',
						fileErrorMsg4860:'<%=rb.getString("ZhiChiIMGHeEXTGeShi")%>'
					},
					ca:{
						fileTip:'<%=rb.getString("ZhiChiPatchGeShi")%>',
						fileFmt:'patch',
						fileErrorMsg:'<%=rb.getString("ZhiZhiChiPATCHWenJian")%>'
					},
					fpga:{
						fileTip:'<%=rb.getString("ZhiChiIMGGeShi")%>',
						fileFmt:'IMG',
						fileErrorMsg:'<%=rb.getString("ZhiZhiChiIMGWenJian")%>'
					},
					ap:{
						fileTip:'<%=rb.getString("ZhiChiIMGGeShi")%>',
						fileFmt:'IMG',
						fileErrorMsg:'<%=rb.getString("ZhiZhiChiIMGWenJian")%>'
					}
				},
				shownxp:false,
				productLabel:'',
				fileTypeTip:'',
				showProduct:true
			}
		},
		methods:{
			init(){
				var vm = this;
				vm.fileTypeTip = vm.difFileObj[enbFileVue.file_type].fileTip;
				axios.post("${ctx}/task/upgrade/getProductType.action",stringify({
                			type: 'all'
                		})).then(function(response){
                			var data = response.data;
					vm.productOptions = data;
				})
				//Cloud版本时，显示可见范围的下拉选项 
				if(isCloudCore == 'true'){
					vm.showToWho = true;
				}else{
					vm.showToWho = false;
				}
				if(enbFileVue.file_type == "ap"){
					vm.showProduct = false;
				}
			},
			importFile(){
				$("#uploadForm_enbFileImport input[name='uploadFile']").click();
			},
			cancel(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
				if(isFormChanged(vm.$refs.enbFileImportForm)){
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						enbFileVue.$refs.slide.hide();
					}).catch(() => {
						
					})
				}else{
					enbFileVue.$refs.slide.hide();
				}
			},
			submit(){
				var vm = this;
				vm.$refs.enbFileImportForm.validate((valid) => {
					if(valid){
						var nxpObj = {
								"CR-B4860/EU" : 'eu',
								"CR-B4860/BU" : 'bu',
								"CR-B4860/RU" : 'ru'
						}
						var files = document.querySelector("#uploadFileImportEnb").files;
						$("#uploadForm_enbFileImport [name=fileSize]").val(files[0].size);
						var pathSplit = vm.importForm.newFileName.split(/\\/);
					    var filename = pathSplit[pathSplit.length - 1];
						$("#uploadForm_enbFileImport [name=newFileName]").val(filename);
						if(enbFileVue.file_type == "ap"){
							$("#uploadForm_enbFileImport [name=product]").val("ap");
						}else{
							$("#uploadForm_enbFileImport [name=product]").val(vm.importForm.product);
						}
						$("#uploadForm_enbFileImport [name=version]").val(vm.importForm.version);
						if(vm.showToWho){
							$("#uploadForm_enbFileImport [name=to_who]").val(vm.importForm.to_who);
						}else{
							$("#uploadForm_enbFileImport [name=to_who]").val('all');
						}
						$("#uploadForm_enbFileImport [name=recommend]").val(vm.importForm.recommend);
						$("#uploadForm_enbFileImport [name=desc]").val(vm.importForm.desc);
						
						if(vm.importForm.product.includes("CR-B4860")){
							$("#uploadForm_enbFileImport [name=fileType]").val(nxpObj[vm.importForm.product]);
						}else{
							$("#uploadForm_enbFileImport [name=fileType]").val(enbFileVue.file_type);
						}
						$('#progressUploadFile').progressbar('setValue', 0);// 将进度条进度置为0
					    $("#winUploadPro").window("open");// 打开进度条窗口
					    intervalGetProgress = window.setInterval(function(){
					    	getUploadProgress();
							var dom = $("#progressUploadFile");
							if(dom.length == 0) clearInterval(intervalGetProgress);
					    }, 1000);// 定时读取进度
					    uploadWithProgress({
					    	url: "${ctx}/cell/version/uploadVersionFile.action",
					    	form: document.querySelector("#uploadForm_enbFileImport"),
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
					            	document.getElementById("uploadForm_enbFileImport").value = "";// 置空文件组件
					            	$.messager.alert(TiShi,"<%=rb.getString("ShangChuanChengGong")%><%=rb.getString("DouHao")%><%=rb.getString("WenJianMD5Zhi")%><%=rb.getString("MaoHao")%>"+data["MD5"]);
					            	enbFileVue.$refs.slide.hide();
					            	enbFileVue.$refs.file_table.refresh();
					            } else {
					            	showMsg('prompt_msg',data["message"]);
					            }
					    	}
					    });
					}
				})
			},
			fileFormatMatch(str,regs){
				var regsArr = regs.toLowerCase().split(",");
				if(str.substring(str.length-6) == 'tar.gz'){
					var suffix = "tar.gz";
				}else{
					var suffix = str.substring(str.lastIndexOf(".")+1).toLowerCase();
				}
				if(regsArr.indexOf(suffix)>-1){
					return true;
				}else{
					return false;
				}
			},
			getVersion(val){
				var vm = this;
				if(vm.productLabel == 'DXDF' && enbFileVue.file_type == 'upgrade'){
					var fileFmt = vm.difFileObj[enbFileVue.file_type].fileFmtYD;
				}else if((vm.productLabel == 'CR-B4860/EU' || vm.productLabel == 'CR-B4860/RU') && enbFileVue.file_type == 'upgrade'){
					var fileFmt = vm.difFileObj[enbFileVue.file_type].fileFmt4860;
				}else{
					var fileFmt = vm.difFileObj[enbFileVue.file_type].fileFmt;
				}
				if(val != '' && vm.fileFormatMatch(val,fileFmt)){
					var pathSplit = val.split(/\\/);
					var filename = pathSplit[pathSplit.length - 1];
					if(filename.substring(filename.length-6) == 'tar.gz'){
						vm.importForm.version = filename.substring(0,filename.length-7);
					}else{
						vm.importForm.version = filename.substring(0,filename.lastIndexOf("."));
					}
					vm.$refs.enbFileImportForm.validateField('version')
				}
			},
			cancelImport(){
				enbFileVue.$refs.tslide.hide();
			}
		},
		watch:{
			"importForm.newFileName":function(val){
				this.getVersion(val)
			},
			"importForm.product":function(val){
				var vm = this;
				vm.productOptions.map(function(item){
					if(item.value == val){
						vm.productLabel = item.name;
					}
				})
			},
			productLabel:function(val){
				var vm = this;
				if(val == "DXDF" && enbFileVue.file_type == 'upgrade'){
					vm.fileTypeTip = vm.difFileObj[enbFileVue.file_type].fileTipYD
				}else if((val == "CR-B4860/EU" || val == "CR-B4860/RU") && enbFileVue.file_type == 'upgrade'){
					vm.fileTypeTip = vm.difFileObj[enbFileVue.file_type].fileTip4860;
				}else{
					vm.fileTypeTip = vm.difFileObj[enbFileVue.file_type].fileTip;
				}
				if(vm.importForm.newFileName != ''){
					vm.$refs.enbFileImportForm.validateField('newFileName');
					vm.getVersion(vm.importForm.newFileName)
				}
			}
		},
		mounted(){
			var vm = this;
			vm.init();
			$("#uploadForm_enbFileImport input[name='uploadFile']").bind("change", function() {
				vm.importForm.newFileName = this.value
			});
			eventBus.$off('import-file').$on('import-file',vm.submit);
			eventBus.$off('cancel-import').$on('cancel-import',vm.cancel);
		}
	}) 
</script>