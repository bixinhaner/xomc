<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#egwLicensePage{
    width:100%;
    height:calc(100% - 10px);
    position: relative;
}
#egwLicensePage .itemMainBoxCls{
	border:1px solid #d5dcec;
	border-radius:10px;
	margin:8px;
	background:#fff;
	height:100%;
}
#egwLicensePage .egwTabPaneHeadCls{
    margin: 25px 0px 0px 40px;
    font-size: 14px;
    font-weight: bold;
    position: relative;
}
#egwLicensePage .egwTabPaneContent{
    margin: 20px 0px 0px 60px;
}
#egwLicensePage .egwTabPaneContent .egwTabPaneItemCls{
    margin-bottom: 10px;
    font-size: 14px;
    display: flex;
}
#egwLicensePage .egwTabPaneContent .egwTabPaneItemCls .egwTabPaneItemLabelCls{
    color: #7A7992;
    width: 180px;
    height: 28px;
    line-height: 28px;
}
#egwLicensePage .egwTabPaneContent .egwTabPaneItemCls .egwTabPaneItemValueCls{
    color: rgba(0, 0, 0, 0.8);
    display: flex;
    align-items: center;
}
#egwLicensePage .egwTabPaneItemValueCls .el-icon-status-execute-success:before{
	color:#67D972;
}
#egwLicensePage .egwTabPaneItemValueCls .el-icon-status-execute-failure:before{
	color:#E88282;
}
#egwLicensePage .licenseFileImportBtn{
    height: 28px;
    display: flex;
    align-items: center;
    padding: 0px 10px;
    border: 1px solid #DFE2EE;
    background: #F6F7FB;
    box-sizing: border-box;
    border-radius: 4px;
    cursor: pointer;
}
#egwLicensePage .dividedLineCls{
    height: 1px;
    background: #DFE2EE;
    width: 100%;
}
#egwLicensePage .tableMainBoxCls{
    height: calc(100% - 340px);
    border:1px solid #DFE2EE;
    box-sizing: border-box;
    margin: 10px 30px;
}
</style>

<div id="egwLicensePage">
	<div class="itemMainBoxCls" v-if="!importLicenseBoxShow">
        <div class="egwTabPaneHeadCls">
            <span class="el-icon el-icon-splitGroup"></span>
            <span><%=rb.getString("XinXi")%></span>

            <div v-show="optBtnShow" class="newIconBoxCls-bt" style="right:55px;top:0px;" @click="syncSubmit" tip='<%=rb.getString("TongBu")%>'>
                <span class="el-icon el-icon-circle-refresh"></span>
            </div>
            <div v-show="optBtnShow" class="newIconBoxCls-bt" style="right:20px;top:0px;" @click="openImportLicenseClick" tip='<%=rb.getString("DaoRu")%>'>
                <span class="el-icon el-icon-operation-import"></span>
            </div>
        </div>
		<div class="egwTabPaneContent">
            <div class="egwTabPaneItemCls">
                <div class="egwTabPaneItemLabelCls">Total Time :</div>
                <div class="egwTabPaneItemValueCls" v-if="egwLicenseData.totalTime == '65535'"><%=rb.getString("Yongjiu")%></div> 
                <div class="egwTabPaneItemValueCls" v-if="egwLicenseData.totalTime != '65535'">
                    {{egwLicenseData.totalTime}}
                    <span v-if="egwLicenseData.totalTime"><%=rb.getString("DanDuTian")%></span> 
                </div> 
            </div>
            <div class="egwTabPaneItemCls" v-if="egwLicenseData.totalTime != '65535'">
                <div class="egwTabPaneItemLabelCls">Remain Time :</div> 
                <div class="egwTabPaneItemValueCls">
                    {{egwLicenseData.remainTime}}
                    <span v-if="egwLicenseData.remainTime"><%=rb.getString("DanDuTian")%></span> 
                </div> 
            </div>
            <div class="egwTabPaneItemCls">
                <div class="egwTabPaneItemLabelCls"><%=rb.getString("XianDingZuiDaJiZhanGuiGe")%> :</div> 
                <div class="egwTabPaneItemValueCls">{{egwLicenseData.maxCount}}</div> 
            </div>
            <div class="egwTabPaneItemCls">
                <div class="egwTabPaneItemLabelCls"><%=rb.getString("XianDingZuiDaIKESAGuiGe")%> :</div> 
                <div class="egwTabPaneItemValueCls">{{egwLicenseData.maxIkeSA}}</div> 
            </div>
        </div>
	</div>
    <div class="itemMainBoxCls" v-if="importLicenseBoxShow">
        <div class="egwTabPaneHeadCls">
            <span>Import License</span>
            <span style="margin-right: 10px;" class="el-icon el-icon-close" @click="closeImportLicenseBox"></span>
        </div>
		<div class="egwTabPaneContent">
            <div class="egwTabPaneItemCls">
                <div class="egwTabPaneItemLabelCls">License File</div>
                <div class="egwTabPaneItemValueCls">
                    <div>{{licenseFileAddFileName}}</div>
                    <div style="margin-left: 10px;">
                        <el-upload ref="licenseFileAddUpload"
                            :before-upload='licenseFileAddBeforeUpload' 
                            :on-success='licenseFileAddCheckFile' 
                            :on-change="licenseFileAddFileChange" 
                            :show-file-list="false" 
                            :action="licenseFileAddUploadFileURL" 
                            :data="licenseFileAddFileParams" 
                            name="licenseFileUploadFile" 
                            :auto-upload="false"
                            accept=".bin">
                            <div class="licenseFileImportBtn" @click="licenseFileAddFileSelect">
                                <span class="el-icon el-icon-operation-import grayIcon"></span>
                                <span style="margin-left: 10px;">Import File</span>
                            </div>
                            <div slot="tip" class="el-upload__tip" v-show="!typeFlag"><%=rb.getString("DaoRuWenJianLeiXingBIN")%></div>
                            <a slot="trigger" ref="licenseFileAddFile_up"></a>
                        </el-upload>
                    </div>
                </div> 
            </div>
            <div class="egwTabPaneItemCls" style="margin-bottom: 30px;">
                <div class="egwTabPaneItemLabelCls"><%=rb.getString("ZhuangTai")%></div> 
                <div class="egwTabPaneItemValueCls">
                    <div v-if="licenseFileAddStatus == '0'">
                        <span class='el-icon el-icon-status-unexecuted'></span>
                        <span style="margin-left: 10px;"><%=rb.getString("WeiZhiXing")%></span>
                    </div>
                    <div v-if="licenseFileAddStatus == '1'">
                        <span class='el-icon el-icon-status-executing'></span>
                        <span style="margin-left: 10px;"><%=rb.getString("ZhengZaiZhiXing")%></span>
                    </div>
                    <div v-if="licenseFileAddStatus == '2'">
                        <span class='el-icon el-icon-status-execute-failure'></span>
                        <span style="margin-left: 10px;"><%=rb.getString("ZhiXingShiBai")%></span>
                    </div>
                    <div v-if="licenseFileAddStatus == '3'">
                        <span class='el-icon el-icon-status-execute-success'></span>
                        <span style="margin-left: 10px;"><%=rb.getString("ZhiXingChengGong")%></span>
                    </div>
                </div> 
            </div>
            <div class="egwTabPaneItemCls" style="margin-bottom: 30px;">
                <el-button @click="activeLicenseClick" :disabled="!licenseFileAddFileName || licenseFileAddStatus == '1' || licenseFileAddStatus == '3'" type="primary">Active Now</el-button>
            </div>
        </div>
        <div class="dividedLineCls"></div>
        <div class="egwTabPaneHeadCls">
            <span>logs</span>
        </div>
        <div class="tableMainBoxCls">
            <el-ctable 
                ref="activeLicenseFileTable" 
                :rownumber="true" 
                :time="6" 
                id="activeLicenseFileTable" 
                :url="activeLicenseFileUrl"
                @load-success="tableLoadSuccess" 
                :query-params="resultsQueryParams" 
                height="100%" 
                pagination="true"
             >
                <el-table-column prop="file_name" label="<%=rb.getString("LicenseWenJian")%>" min-width="120"></el-table-column>
                <el-table-column prop="task_status" label="<%=rb.getString("ZhuangTai")%>" min-width="80">
                    <template slot-scope="scope">
                        <div v-html="resultTableStatus(scope.row.task_status)"></div>
                    </template>
                </el-table-column>
                <el-table-column prop="task_result" label="<%=rb.getString("JieGuo")%>" min-width="80"> 
                    <template slot-scope="scope">
                        <div v-html="resultTableResult(scope.row.task_result)"></div>
                    </template>
                </el-table-column>
                <el-table-column prop="start_time" label="<%=rb.getString("KaiShiShiJian")%>" min-width="100"></el-table-column>
                <el-table-column prop="end_time" label="<%=rb.getString("JieShuShiJian")%>" min-width="100"></el-table-column>
            </el-ctable>
        </div>
	</div>
</div>

<script>
var egwLicensePage = new Vue({
	el: '#egwLicensePage', 
	data() {
		var vm = this;
		return {
            egwCode:'',
            egwSn:'',
            rowData:{
                generation:'',
            },
            egwLicenseData:{
                totalTime:'',
                remainTime:'',
                maxCount:'',
                maxIkeSA:'',
            },
            importLicenseBoxShow:false,

            licenseFileAddFileParams:{},
            licenseFileAddFileName:'',
            licenseFileAddStatus:'',
            licenseFileAddUploadFileURL:'',
            activeLicenseFileUrl:'',
            resultsQueryParams:{
				timeZone:timeZone,
				egwCode:'',
			},
			typeFlag: true,
		};
	},
	computed: {
        optBtnShow() {
			return writableMap['CODE_EGW'] == true;
		},
    },
	methods: {
        // 初始化
		init(row,code,sn,status){
		    var vm =this;
			vm.egwCode = code;
            vm.egwSn = sn;
            vm.rowData = row;
            vm.resultsQueryParams.egwCode = code;
            vm.getInformationData();
		},
        getInformationData(){
            var vm = this,
                params = {
                   egwCode: vm.egwCode
                };
            axios.post("${ctx}/egw/config/getInformation.action",stringify(params)).then(function(response){
                var data = response.data;
                if(data){
                    vm.egwLicenseData.totalTime = data.totalTime ? data.totalTime : '';
                    vm.egwLicenseData.remainTime = data.remainTime ? data.remainTime : '';
                    vm.egwLicenseData.maxCount = data.maxCount ? data.maxCount : '';
                    vm.egwLicenseData.maxIkeSA = data.maxIkeSA ? data.maxIkeSA : '';
                }
            });
        },
        // 同步
        syncSubmit(){
            var vm = this,
                urls='${ctx}/egw/config/refreshConfig.action',
                params={
                    egwCode: vm.egwCode,
                    refreshType: 'LICENSE'
                };
            axios.post(urls,stringify(params)).then(res=>{
                var data = res.data;
                if(data["success"]){
                    vm.$message({
                        message: '<%=rb.getString("MingLingYiXiaFa")%>',
                        type:'success',
                    });
                }else{
                    vm.$message.error(data["message"])
                }
            })
        },
        // 打开导入license弹框
        openImportLicenseClick(){
            var vm = this,
                randomCode = Math.random().toString();
            vm.importLicenseBoxShow = true;
            vm.activeLicenseFileUrl = '${ctx}/egw/license/manage/getTaskList.action?randomCode='+randomCode;
            vm.refreshLicenseData();
        },
        closeImportLicenseBox(){
            var vm = this;
            vm.importLicenseBoxShow = false;
        },
        licenseFileAddBeforeUpload(file){
			var vm = this, 
				urls = '${ctx}/egw/license/manage/importFile.action',
				FileName = file.name,
				fd = new FormData(),
				config = {
					headers: { 'Content-Type': 'multipart/form-data' }
				};
			fd.append('uploadFile',file); //文件流
			fd.append('FileName',FileName);//文件名
            fd.append('egwCode',vm.egwCode);//文件名
			axios.post(urls,fd,config).then(function(res){
				let data = res.data;
				if(data){
                    vm.$message({
						type: 'success',
						message: '<%=rb.getString("ChengGong")%>'
					});
                    vm.refreshLicenseData();
				}else{
					vm.$message.error(data["msg"])
				}
			})
			return false;
		},
		licenseFileAddCheckFile(res,file){    //发送请求，校验device文件内容 
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
				vm.licenseFileCloseAddFileSelect();
			}else{
				vm.$message({
					type: 'error',
					message: res.msg
				});
			}
			//修改已选择文件状态  
			var fileList = vm.$refs.licenseFileAddUpload.uploadFiles;
			fileList.forEach(function(file){
				file.status = 'ready';
			})
		},
		// 移除导入文件
		licenseFileCloseAddFileSelect(){
			var vm = this;
			vm.licenseFileAddFileName = '';
			vm.$refs.licenseFileAddUpload.clearFiles();
		},
		// 选择文件
		licenseFileAddFileSelect(){  
			var vm =this;
			vm.$refs.licenseFileAddUpload.clearFiles();
			vm.$refs['licenseFileAddFile_up'].click();
		},
		/**
		* 选择文件后，校验格式，并赋值页面显示 
		* @param file{object}   文件信息
		* @param fileList{Array}  文件列表
		*/ 
		licenseFileAddFileChange(file,fileList){ 
			var vm = this;
			const typeFlag = file.name.substr(file.name.lastIndexOf("."))  === '.bin';
            vm.typeFlag = typeFlag;
			if(typeFlag){
                vm.licenseFileAddStatus = '0';
                vm.$refs.licenseFileAddUpload.submit();
			}else {
				vm.licenseFileAddFileName = '';
			}
		},
        // 激活 license
        activeLicenseClick(){
            var vm = this,
                urls='${ctx}/egw/license/manage/addTask.action',
                params={
                    egwCode: vm.egwCode,
                };
            axios.post(urls,stringify(params)).then(res=>{
                var data = res.data;
                if(data["success"]){
                    vm.$message({
                        message: '<%=rb.getString("ChengGong")%>',
                        type:'success',
                    });
                }else{
                    vm.$message.error(data["message"])
                }
            })
        },
        // 刷新 license信息
        refreshLicenseData(){
            var vm = this,
                urls='${ctx}/egw/license/manage/getFileInfo.action',
                params={
                    egwCode: vm.egwCode,
                };
            axios.post(urls,stringify(params)).then(res=>{
                var data = res.data;
                if(data){
                    vm.licenseFileAddFileName = data.file_name ? data.file_name : '';
                    vm.licenseFileAddStatus = data.active_status ? data.active_status : '';
                }
            })
        },
        // 表格数据请求回调
        tableLoadSuccess(){
            var vm = this;
            vm.refreshLicenseData();
        },
        resultTableStatus(cellValue){
			var statusObj = {
				'1':'<%=rb.getString("DengDai")%>',	
				'2':'<%=rb.getString("JinXingZhong")%>',	
				'4':'<%=rb.getString("YiJieShu")%>',	
				'':'',	
			}
			return statusObj[cellValue];
		},
		resultTableResult(cellValue){
			var resultObj = {
				'1' : '<%=rb.getString("ChengGong")%>',
				'3' : '<%=rb.getString("ShiBai")%>',
				'' : '',
			}
			return resultObj[cellValue];
		},
	},
	mounted() {
		eventBus.$off("egw-data").$on("egw-data",this.init)
	}
});

</script>
