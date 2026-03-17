<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<style>
#egwUpgradePages{
    width:100%;
    height:calc(100% - 10px);
    position: relative;
}
#egwUpgradePages .itemMainBoxCls{
	border:1px solid #d5dcec;
	border-radius:10px;
	margin:8px;
	background:#fff;
	height:100%;
}
#egwUpgradePages .itemMainContent{
     padding: 40px 0px 0px 40px;
     box-sizing: border-box;
}
#egwUpgradePages .importantBtn{
    height: 28px;
    line-height: 28px;
    padding: 0px 10px;
    margin-left: 10px;
}
.itemMainBoxFooter{
    display: flex;
    align-items: center;
    border-top : 1px solid #E9E9E9;
	height:48px;
    box-sizing: border-box;
    width: calc(100% - 50px);
    position: absolute;
    bottom: 0px;
}
.egwSettingAddDialogCls .el-input__suffix{
    height: 26px;
    display: flex;
    align-items: center;
}
</style>
<div id="egwUpgradePages">
    <div class="itemMainBoxCls">
        <div class="itemMainContent">
            <el-form :model='ruleForm' ref="ruleForm" :rules="rules" label-position="left" label-width="180px">
                <div style='margin-bottom: 10px;'>
                    <span style='display: inline-block; width: 180px;font-size:14px;'><%=rb.getString("DangQianBanBen")%></span>
                    <span>{{deviceData.softwareVersion}}</span>
                </div>
                <el-form-item prop="fileId" label="<%=rb.getString("ShengJiBanBen")%>">
                    <el-select v-model="ruleForm.fileId" style="padding-top:7px;">
                        <el-option v-for="item in upgradeFileList" :label="item.version" :value="item.id" :key="item.id"></el-option>
                    </el-select>
                    <el-button class="importantBtn" @click="addEgwUpgradeFile">
                        <i class="el-icon el-icon-operation-import" style="position:relative;top:2px;"></i>
                         Import file
                    </el-button>
                </el-form-item>
            </el-form>
        </div>
         <div class='itemMainBoxFooter'>
            <el-button type="primary" @click="submit" style="margin-left:20px;"><%=rb.getString("LiJiShengJi")%></el-button>
        </div>
    </div>
	
     <!-- 证书导入 弹窗 -->
	<el-dialog id="CertImport" class="egwSettingAddDialogCls" title="<%=rb.getString("DaoRu")%>" top="30vh" width="480px" :visible.sync="egwUpgradeFileImportShow" class="importCard" :close-on-click-modal="false" append-to-body @close="closeImportParams">		
		<el-form label-position="top" ref="egwUpgradeFileImportForm" :model='egwUpgradeFileImportForm' :rules='egwUpgradeFileImportRules'>     		     			            
        	<el-form-item label="<%=rb.getString("DaoRuWenJian")%>" label-width="110px" prop="FileName" style="margin-bottom:20px;">
                <span slot="label" class="labelSlotCls">
                    <%=rb.getString("DaoRuWenJian")%>
                    <span style="color:rgba(0, 0, 0, 0.32)">( Only .rpm is supported )</span>
                </span>
               	<el-upload 
             		ref="egwUpgradeFileUpload"
             		:before-upload='egwUpgradeFileBeforeUpload' 
             		:on-success='egwUpgradeFileCheckFile' 
             		:on-change="egwUpgradeFileFileChange"  
             		:show-file-list=false 	                  		
				    :action="egwUpgradeFileImportForm.uploadFileUrl" 
				    :data="egwUpgradeFileImportParams" 
				    name="uploadFile" 
				    :accept="'.rpm'"
				    :auto-upload="false">
					<el-input v-model="egwUpgradeFileImportForm.FileName" placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' style="width:320px;">
						<a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="egwUpgradeFileImportFileSelect"></a>
					</el-input>
					<a slot="trigger" ref="file_up"></a>
				</el-upload>	
         	</el-form-item>
            <el-form-item label="Version" label-width="110px">
         		<el-input v-model="egwUpgradeFileImportForm.version" style="width:320px;"></el-input>
         	</el-form-item>   
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="egwUpgradeFileUploadParams"><%=rb.getString("QueDing")%></el-button>
              <el-button @click="closeImportParams"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
</div>

<script type="text/javascript">
	var egwUpgradePages = new Vue({
	    el: '#egwUpgradePages',
	    data() {
	    	var vm = this,
	    		validateVersion = (rule,value,callback) => {
	    			if(value === '' || value === null || value === undefined){
						callback(new Error('<%=rb.getString("QingXianXuanZeWenJian")%>'))
					}else{
						callback();
					}
				},
                fileNameValidate = function(rule,value,callback) {
                    if(value) {
                        if(value.length>100) {
                            callback('<%=rb.getString("WenJianMingBuNengChaoGuoYaoQiu")%>');
                        }else if(!fileFormatMatch(value,"rpm")){
                            callback('<%=rb.getString("ZhIZhiChiRPMWenJian")%>');
                        }else {
                            callback();
                        }
                    }else {
                        callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
                    }
                };
	    	return {
                egwCode:'',
                egwSn:'',
                deviceData:'',
	    		ruleForm:{
                    taskName: '',
                    productType:'eGW',
	    			fileId:'',
                    currentVersion:''
                },
                rules:{
                   fileId:[
						{validator: validateVersion,trigger:'change'}
					]
                },
                upgradeFileList:[],
                egwUpgradeFileImportParams:{}, 
                egwUpgradeFileImportShow:false,
                egwUpgradeFileImportForm: {
                    file:'',
                    uploadFileUrl: '',
                    errMassageShow:false,
                    errMassage:'',
                    FileName:'',
                    version:'',
                },	   
                egwUpgradeFileImportRules: {
                    FileName:[
						{validator: fileNameValidate}
					],
                },
                fileErrorData:'',
                difFileObj:{
                    upgrade:{
                        fileTip:'<%=rb.getString("RPMWenJianTiShi")%>',
                        fileFmt:'rpm',
                        fileErrorMsg:'<%=rb.getString("ZhIZhiChiRPMWenJian")%>',
                    }
                },
	    	}
	    },
        watch:{
            "egwUpgradeFileImportForm.FileName":function(val){
                var vm = this;
                this.getVersion(val);
            },
        },
	    methods: {
            // 初始化
            init(row,code,sn,status){
                var vm =this;
                vm.egwCode = code;
                vm.egwSn = sn;
                vm.deviceData = row;
                vm.getFileListData();
            },
            // 获取文件下拉数据
            getFileListData(){
                var vm = this;
                axios.post('${ctx}/egw/softwareFile/queryAllSoftwareFilePageList.action').then(function(response){
					var data = response.data;
					
					if(data && data.length > 0){
						vm.upgradeFileList = data.map(function(item){
				   			if (item){
				   				return {version:item.version,id:item.id}
				   			}
				   		})
					}else{
						vm.upgradeFileList = [];
					}
				}).catch(function(error){})
            },
            // 升级文件导入
            addEgwUpgradeFile(){
                var vm = this;
                vm.egwUpgradeFileImportShow = true;
            },
            // 提交
			submit(){
				var vm = this,
                    urls = '${ctx}/egw/softwareUpgrade/addTask.action',
                    params = {};
                var curTaskName = '<%=rb.getString("RuanJianShengJi")%>' + '_' + user_code + '_' + formatDate(new Date(gloableTime)); 
				vm.$refs.ruleForm.validate((valid) => {
                    if(valid){
                        var params = {};
                        params.timeZone = timeZone;
                        params.egwCodes = vm.egwCode;
                        params.taskName = curTaskName;
                        params.selectAll = 'false';
                        params.status = 'active';
                        params.fileId = vm.ruleForm.fileId;
                        axios.post(urls,stringify(params)).then(function(response){
                            var data = response.data;
                            if(data["success"]){
                                vm.$message({
                                    message:'<%=rb.getString("ChengGong")%>',
                                    type:'success',
                                })
                                vm.closeLinkSetting();
                            }else{
                                vm.$message.error(data["message"])
                            }
                        }).catch(function(error){})
                    }else{
                        return false;
                    }
                })
				
			},
            // 关闭配置页面
            closeLinkSetting(){
                var vm = this;
                egwMonitor.$refs.egwSettingPage.hide();
            },
            // 选择文件
            egwUpgradeFileImportFileSelect(){  
                var vm =this;
                vm.$refs.egwUpgradeFileUpload.clearFiles();
                vm.$refs['file_up'].click();
            },
            /**
            * 文件上传之前
            * @param file{object}   文件信息
            */ 
            egwUpgradeFileBeforeUpload(file){
                var vm = this, 
                    urls = '${ctx}/egw/softwareFile/uploadSoftwareFile.action', 
                    fileName = file.name,
                    fileSize = file.size,
                    fd = new FormData(),
                    config = {
                        headers: { 'Content-Type': 'multipart/form-data' },
                        onUploadProgress:(ev)=>{
                            if(ev.lengthComputable || ev.event.lengthComputable) {
                                var total = ev.total,
                                    loaded = ev.loaded,
                                    percent = 100*loaded/total;
                                $('#progressUploadFile').progressbar('setValue', percent.toFixed(2));
                            }
                        }
                    };
                
                fd.append('uploadFile',file); //文件流
                fd.append('fileSize',fileSize);//文件大小
                fd.append('description','');//描述
                fd.append('productType',vm.ruleForm.productType);
                fd.append('version',vm.egwUpgradeFileImportForm.version);
                vm.fileErrorData = vm.$refs.egwUpgradeFileUpload.uploadFiles[0];
                $('#progressUploadFile').progressbar('setValue', 0);// 将进度条进度置为0
			    $("#winUploadPro").window("open");// 打开进度条窗口
                axios.post(urls,fd,config).then(function(response){
                    var data = response.data
                    $("#winUploadPro").window("close");// 关闭进度条窗口
                    if(data["success"]){
                        vm.fileErrorData = '';
                        $.messager.alert('<%=rb.getString("TiShi")%>','<%=rb.getString("ShangChuanChengGong")%><%=rb.getString("DouHao")%><%=rb.getString("WenJianMD5Zhi")%><%=rb.getString("MaoHao")%>'+data["message"]);
                        vm.getFileListData();
                        vm.closeImportParams();	
                    }else{
                        vm.$message.error(data["message"]);
                    }
                })				
                return false;
            },
            // 关闭导入弹出框
            closeImportParams(){
                var vm = this,
                    params = {
                        uploadFileUrl: '',
                        errMassageShow:false,
                        errMassage:'Only .rpm is supported',
                        FileName:'',
                        version:'',
                        file:'',
                    };
                Object.assign(vm.egwUpgradeFileImportForm,params);		
                vm.$refs.egwUpgradeFileUpload.clearFiles();
                vm.egwUpgradeFileImportShow = false;
            },
            //导入
            egwUpgradeFileCheckFile(res,file){  
                var vm = this;
                if(res.success){
                    if(res.suc_count>0){
                        vm.$message({
                            type: 'success',
                            message: '<%=rb.getString("ChengGong")%>'
                        });
                    }else {
                        vm.$message({
                            type: 'warning',
                            message: '<%=rb.getString("ShiBai")%>'
                        });
                    }				
                    vm.closeImportParams();
                }else{
                    vm.$message({
                        type: 'error',
                        message: res.message
                    });
                }
                //修改已选择文件状态  
                var fileList = vm.$refs.egwUpgradeFileUpload.uploadFiles;
                fileList.forEach(function(file){
                    file.status = 'ready';
                })
            },
            /**
            * 选择文件后，校验格式，并赋值页面显示 
            * @param file{object}   文件信息
            * @param fileList{Array}  文件列表
            */ 
            egwUpgradeFileFileChange(file,fileList){ 
                var vm = this,typeFlag;

                var vm = this;
                vm.fileErrorData = '';
                vm.egwUpgradeFileImportForm.FileName = file.name;
            },
            /*确定导入*/
            egwUpgradeFileUploadParams() {
                var vm = this;
                var vm = this;
                vm.$refs.egwUpgradeFileImportForm.validate((valid) => {
                    if(valid){
                       if(vm.fileErrorData){
                            vm.$refs.importFile.uploadFiles.push(vm.fileErrorData);
                        }
                        vm.$refs.egwUpgradeFileUpload.submit();
                    }
                })
            },
            getVersion(val){
                var vm = this;

                var fileFmt = this.difFileObj['upgrade'].fileFmt;

                if(val != '' && vm.fileFormatMatch(val,fileFmt)){
                    var pathSplit = val.split(/\\/);
                    var filename = pathSplit[pathSplit.length - 1];
                    if(filename.substring(filename.length-6) == 'tar.gz'){
                        vm.egwUpgradeFileImportForm.version = filename.substring(0,filename.length-7);
                    }else{
                        vm.egwUpgradeFileImportForm.version = filename.substring(0,filename.lastIndexOf("."));
                    }
                    vm.$refs.egwUpgradeFileImportForm.validateField('version')
                }
            },
            // 文件校验
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
	    },
		mounted(){
            eventBus.$off("egw-data").$on("egw-data",this.init)
	    }
	});
	
</script> 