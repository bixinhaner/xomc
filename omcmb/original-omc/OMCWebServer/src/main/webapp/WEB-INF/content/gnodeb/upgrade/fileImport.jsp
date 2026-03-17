<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<!DOCTYPE html>
<html>
<head>
	<style>
		#gnb_file_import .flex-inline {
			display: flex;
			align-items: center; 
		}
		#gnb_file_import .flex-inline .el-form-item__content {
			margin-left: 0px !important;
		}
		.info{
			padding: 20px;
		}
		.info input{
			min-height: 24px;
		}
	</style>
</head>
<body>
  <div id="gnb_file_import" class="info">
    <el-form ref="gnbFileImportForm" :model="form" :rules="rules" label-width="120px">
		<el-form-item label="Product Type" class="flex-inline" prop="productValue">
			<el-select v-model='form.productValue'>
				<el-option v-for='item in productOptions' :label="item.name" :value="item.value"></el-option>
			</el-select>
		</el-form-item>
		<el-form-item label="File Name" class="flex-inline" prop="newFileName">
			<el-input v-show="editable" style='width:400px;' v-model="form.newFileName" :disabled="true">
				<i @click="importFile" slot='suffix' style='display:inline-block;width:28px;height:28px;margin:4px -9px 0 0;' class='el-icon el-icon-operation-import'></i>
			</el-input>
			<span v-show="editable" style='color:#999;margin-left:10px;'>{{fileTypeTip}}</span>

			<span v-show="!editable">{{form.newFileName}}</span>
		</el-form-item>
		<el-form-item label="Version" class="flex-inline" prop="version">
			<el-input v-model="form.version" :disabled="readonly"></el-input>
		</el-form-item>
		<el-form-item label="Recommend" class="flex-inline" prop='recommend'>
			<el-select v-model="form.recommend" :disabled="readonly">
				<el-option label="Yes" value="1"></el-option>
				<el-option label="No" value="0"></el-option>
			</el-select>
		</el-form-item>
		<el-form-item label="Description" prop='desc'>
			<el-input v-model="form.desc" type="textarea" style="width:60%;" :rows="8" :disabled="readonly"></el-input>
		</el-form-item>
    </el-form>
  </div>
  <%-- 表单-上传基站列表文件 --%>
  <form enctype="multipart/form-data" method="post" id="uploadForm_gnbFileImport" style="display: none;">
		<input name="uploadFile" type="file" id="uploadFileImportGnb">
		<input name="newFileName" id="newFileName" value="" hidden="true">
		<input name="fileSize" value="" hidden="true">
		<input name="fileType" id="fileType" value="" type="hidden"/>
		<input name="product" value="" type="hidden"/>
		<input name="version" value="" type="hidden"/>
		<input name="to_who" value="all" type="hidden"/>
		<input name="desc" value="" type="hidden"/>
		<input name="md5" value="" type="hidden"/>
		<input name="recommend" value="" type="hidden"/>
		<input name="isGnb" value="1" type="hidden"/>
  </form>

  <script type="text/javascript">
  	new Vue({
  		el: '#gnb_file_import',
  		data() {
  			var vm = this,
			  	validateFilePath = (rule,value,callback) => {
					var reg = vm.difFileObj['upgrade'].fileFmt;
					var message = vm.difFileObj['upgrade'].fileErrorMsg;
					
					if(value == ""){
						callback(new Error('<%=rb.getString("QingXianXuanZeWenJian")%>'))
					}else if(!vm.fileFormatMatch(value,reg)){
						callback(new Error(message))
					}else{
						var pathSplit = value.split(/\\/),
							filename = pathSplit[pathSplit.length - 1];
						
						if(filename.length>100) {
							callback('<%=rb.getString("WenJianMingBuNengChaoGuoYaoQiu")%>');
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
							callback();
						}
					}else {
						callback('<%=rb.getString("QingShuRuWenJianBanBen")%>');
					}
				};

  			return {
  				form: {
					versionId: '',
					uploadFile: '',
					newFileName: '',
					fileSize: '',
					fileType: 'upgrade',
					version: '',
					desc: '',
					recommend: '0',
					md5: '',
					isGnb: 1,
					toWho: 'all',
					productValue:''
				},
				rules:{
					newFileName:[
						{validator: validateFilePath}
					],
					version:[
						{validator: versionValidate}
					],
					recommend:[
						{required:true,message:'<%=rb.getString("ShuRuBiTianXiang")%>',trigger:'change'}
					]
				},
				fileTypeTip: '<%=rb.getString("ZhiZhiChiIMGHeEXTGeShi")%>',
				productOptions: [],
				difFileObj:{
					upgrade:{
						fileTip:'<%=rb.getString("ZhiChiIMGGeShi")%>',
						fileTipYD:'<%=rb.getString("ShengJiZhiChiGeShi")%>',
						fileFmt:'IMG,ext',
						fileFmtYD:'tar.gz',
						fileErrorMsg:'<%=rb.getString("ZhiChiIMGHeEXTGeShi")%>',
						fileErrorMsgYD:'<%=rb.getString("ShengJiZhiZhiChiWenJian")%>'
					}
				},
				opType: 'add'
  			};
  		},
		computed: {
			readonly() {
				var vm = this,
					bool = false;
				
				if(vm.opType == 'info') bool = true;

				return bool;
			},
			editable() {

				return this.opType == 'add';
			}
		},
		watch:{
			"form.newFileName":function(val){
				this.getVersion(val)
			},
		},
  		methods: {
			init(params) {
				var vm = this,
					type = params.type||'add',
					row = params.row;
			    
				vm.opType = type;
				
				axios.post("${ctx}/task/upgrade/getProductType.action?isGnb=1").then(function(res){
	            	var data = res.data;
			   		
					if(data.length > 0){
						vm.productOptions = data;
						//数组第一条数据为默认的产品类型
						if(type == 'add'){
							vm.form.productValue = vm.productOptions[0].value;
						}
					}
				});
				
				if(type == 'edit') {
					vm.form.fileType = '0';
				}
				if(type != 'add'){
					var productValue = row.productValue;
					Object.assign(vm.form,{
						versionId: row.id,
						newFileName: row.file_name,
						version: row.version,
						productValue: productValue,
						desc: row.desc,
						recommend: row.recommend
					});
				}
				initForm(vm.$refs.gnbFileImportForm);
				
			},
			importFile(){
				$("#uploadForm_gnbFileImport input[name='uploadFile']").click();
			},
			submit(){
				var vm = this;
				
				if(vm.opType == 'add') {
					vm.saveAddFile();
				}else {
					vm.saveModifyFile();
				}
			},
			cancel(){
				var vm = this;
				if(vm.opType !='info' ){
					var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
					if(isFormChanged(vm.$refs.gnbFileImportForm)){
						vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
							confirmButtonText:'<%=rb.getString("QueDing")%>',
							cancelButtonText:'<%=rb.getString("QuXiao")%>',
							type:'warning',
							closeOnClickModal:false
						}).then(() => {
							gnbFileVue.$refs.slide.hide();
						}).catch(() => {
							
						})
					}else{
						gnbFileVue.$refs.slide.hide();
					}
				}else{
					gnbFileVue.$refs.slide.hide();
				}
				
			},
			saveAddFile() {
				var vm = this;

				vm.$refs.gnbFileImportForm.validate((valid) => {
					if(valid){
						var files = document.querySelector("#uploadFileImportGnb").files;
						$("#uploadForm_gnbFileImport [name=fileSize]").val(files[0].size);
						var pathSplit = vm.form.newFileName.split(/\\/);
					    var filename = pathSplit[pathSplit.length - 1];

						$("#uploadForm_gnbFileImport [name=newFileName]").val(filename);
						$("#uploadForm_gnbFileImport [name=product]").val(vm.form.productValue);
						$("#uploadForm_gnbFileImport [name=version]").val(vm.form.version);
						$("#uploadForm_gnbFileImport [name=recommend]").val(vm.form.recommend);
						$("#uploadForm_gnbFileImport [name=desc]").val(vm.form.desc);
						$("#uploadForm_gnbFileImport [name=fileType]").val(vm.form.fileType);
						
						$('#progressUploadFile').progressbar('setValue', 0);// 将进度条进度置为0
					    $("#winUploadPro").window("open");// 打开进度条窗口
					    intervalGetProgress = window.setInterval(function(){
					    	getUploadProgress();
							var dom = $("#progressUploadFile");
							if(dom.length == 0) clearInterval(intervalGetProgress);
					    }, 1000);// 定时读取进度

					    uploadWithProgress({
					    	url: "${ctx}/cell/version/uploadVersionFile.action",
					    	form: document.querySelector("#uploadForm_gnbFileImport"),
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
								gnbFileVue.$refs.slide.hide();
								gnbFileVue.$refs.file_table.refresh();
					    		if (data["MD5"]) {
					            	$("#upgradeSubmit").addClass("forbidden");
					            	document.getElementById("uploadForm_gnbFileImport").value = "";// 置空文件组件
					            	$.messager.alert(TiShi,'<%=rb.getString("ShangChuanChengGong")%><%=rb.getString("DouHao")%><%=rb.getString("WenJianMD5Zhi")%><%=rb.getString("MaoHao")%>'+data["MD5"]);
					            	
					            } else {
					            	showMsg('prompt_msg',data["message"]);
					            }
					    	}
					    });
					}
				})
			},
			saveModifyFile() {
				var vm = this;
				
				vm.$refs.gnbFileImportForm.validate((valid) => {
					if(valid){
						var params = {
								versionId : vm.form.versionId,
								product: vm.form.productValue,
								fileName: vm.form.newFileName,
								version: vm.form.version,
								recommend: vm.form.recommend,
								desc: vm.form.desc,
								fileType: vm.form.fileType,
								isGnb: 1,
								toWho: vm.form.toWho
							};
						
						axios.post("${ctx}/cell/version/goModifyDeviceVersionFileInfo.action",stringify(params)).then(function(response){
							var data = response.data;
							if(data["success"]){
								vm.$message({
		    						message: '<%=rb.getString("ChengGong")%>',
		    						type: 'success',
		    					})
                                gnbFileVue.$refs.file_table.refresh();
								gnbFileVue.$refs.slide.hide();
							}else{
								vm.$message.error(data["message"])
							}
						})
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
				var fileFmt = this.difFileObj['upgrade'].fileFmt;
				
				if(val != '' && this.fileFormatMatch(val,fileFmt)){
					var pathSplit = val.split(/\\/);
					var filename = pathSplit[pathSplit.length - 1];
					if(filename.substring(filename.length-6) == 'tar.gz'){
						this.form.version = filename.substring(0,filename.length-7);
					}else{
						this.form.version = filename.substring(0,filename.lastIndexOf("."));
					}
					this.$refs.gnbFileImportForm.validateField('version')
				}
			}
  		},
		mounted() {
			var vm = this;

			$("#uploadForm_gnbFileImport input[name='uploadFile']").bind("change", function() {
				vm.form.newFileName = this.value
			});

			eventBus.$off('import-init').$on('import-init',vm.init);
			eventBus.$off('save-imort-file').$on('save-imort-file',vm.submit);
			eventBus.$off('cancel-import').$on('cancel-import',vm.cancel);
		}
  	})
  </script>
</body>
</html>