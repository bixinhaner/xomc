<%@ page contentType="text/html;charset=UTF-8" %>
    <%@ include file="/common/taglibs.jsp" %>
    <%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%
	UserInfo ui = (UserInfo) session.getAttribute(ComConstants.SESSION_KEY);
%>

        <style>
           .tslide .el-card__header i{
                font-size: 16px !important;
                margin-top: 0px !important;
                position:unset
            }
            #viewPciTaskEnb .group-title {
                margin-bottom: 20px
            }
            
            #viewPciTaskEnb .slide-content {
                padding: 0px !important;
                border: none !important
            }
            
            #viewPciTaskEnb .power-system {
                border-bottom: #E9E9E9 solid 1px
            }
            
            #viewPciTaskEnb .power-system li {
                margin-left: 50px;
                margin-bottom: 15px
            }
            
            #viewPciTaskEnb .power-system li lable {
                color: #9E9E9E;
                width: 150px;
                font-size: 14px;
                display: inline-block
            }
            
            #viewPciTaskEnb {
                font-size: 14px;
                padding: 20px
            }
            
            #viewPciTaskEnb .title-text:after {
                content: none
            }
            
            #viewPciTaskEnb .plr15 {
                padding: 0 15px !important
            }
            
            #viewPciTaskEnb .plr15 .el-form-item__label {
                width: 113px;
                line-height: 27px
            }
            
            #viewPciTaskEnb .ml20 {
                margin-left: 20px
            }
            
            #viewPciTaskEnb .w200 {
                width: 200px
            }
            
            #viewPciTaskEnb .mt20 {
                margin-top: 20px
            }
            
            #viewPciTaskEnb .w150 {
                width: 150px
            }
            
            #viewPciTaskEnb .w120 {
                width: 120px
            }
            
            #viewPciTaskEnb .clor9 {
                color: #999999
            }
            
            #viewPciTaskEnb .el-icon-operation-import:before {
                position: relative;
                top: 4px
            }
            
            #viewPciTaskEnb .footer {
                width: 100%;
                overflow: overlay;
                border-right: none;
                border-left: none;
                border-bottom: none;
                position: absolute;
                bottom: 0px;
                left: 0px;
                height: 48px;
                line-height: 48px;
                background: #ffffff
            }
            
            .footer .lnkbuttonGroup {
                margin-left: 48px
            }
            
            .el-form-item__error,
            .el-upload__tip {
                margin-left: 110px
            }
            
            .el-radio-group {
                line-height: 28px;
                margin-top: 3px
            }
            
            #viewPciTaskEnb .el-form-item {
                margin-bottom: 22px
            }
            
            .el-icon-circle-info:before {
                color: #ccc;
                font-size: 18px;
                position: relative;
                top: 2px;
                margin-right: 5px
            }
            
            .notice {
                margin: 10px 0px;
                font-size: 12px;
                color: #999
            }
            
            .ml30 {
                margin-left: 30px
            }
            
            .p12Box .el-form-item__label {
                width: 70px !important
            }
            
            .el-card__header {
                border: none
            }
            
            .el-card__body {
                padding: 0px !important;
                background: #ffffff
            }
            
            .el-silde {
                right: 0px
            }
            
            .ml5 {
                margin-left: 5px
            }
            
            .mt5 {
                margin-top: 5px
            }
            
            .mr5 {
                margin-right: 5px
            }
            
            .f14 {
                font-size: 14px;
                
            }
            .importBox-fileSlect{
                margin-top: 0px !important
            }
        </style>
        <div id='viewPciTaskEnb'>
            <el-form label-position="left" ref="sasImportForm" :model='sasImportForm' :rules='rules'>
                <div class="plr15">
                    <el-form-item label="<%=rb.getString("ExcelFile")%> ">
                        <el-upload :on-success='checkFile' :on-change="fileChange" :show-file-list=false ref="uploadCert" :action="sasImportForm.uploadFileURL"
                            :auto-upload="false">
                            <el-input :readonly="true" :value=fileName class="w200">
                                <a slot="suffix" class="el-icon el-icon-operation-import importBox-fileSlect" @click="fileSelect"></a>
                            </el-input>
                            <span class="clor9">.xls/.xlsx</span>
                            <div slot="tip" class="el-form-item__error" v-show="!typeFlag">
                                <%=rb.getString("ZhiZhiChiExeclFile")%> 
                            </div>
                            <div slot="tip" class="el-form-item__error" v-show="selectFlag">
                                <%=rb.getString("QingXianXuanZeWenJian")%>
                            </div>
                            <a slot="trigger" ref="file_up"></a>
                            <span class="ml30" style="cursor: pointer"><i class="el-icon el-icon-common-download"></i><u @click='exportTemp'> <%=rb.getString("XiaZaiShiLiMoBan")%></u> </span>
                        </el-upload>

                    </el-form-item>

                    <el-form-item label="User ID" prop="userid">
                        <el-input v-model="sasImportForm.userid" class="w200" maxlength='50'></el-input>
                    </el-form-item>
                </div>
            </el-form>
            <div class="footer">
                <div class="lnkbuttonGroup">
                    <el-button type="primary" @click="addEdit">
                        <%=rb.getString("QueDing")%>
                    </el-button>
                    <el-button @click="close">
                        <%=rb.getString("QuXiao")%>
                    </el-button>
                </div>
            </div>
        </div>
        <script>
            var addEnb = new Vue({
                el: '#viewPciTaskEnb',
                data() {
                    let useridValidator = (rule, value, cb) => {
                        if (value === '') {
                            cb('User ID<%=rb.getString("BuNengWeiKong") %>');
                        } else {
                            cb()
                        }
                    };
                    return {
                        sasImportForm: {
                            userid: '',
                            excelFile:'',
                            uploadFileURL: '${ctx}/cell/SAS/importInstallParam.action',
                        },
                        selectFlag: false,        //标识是否选择了文件 
                        typeFlag: true,
                        fileName: '',
                        rules: {
                            userid: [
                                { validator: useridValidator },
                            ],
                        },
                        deviceType:'',
                    }
                },
                methods: {
                    /**
                    * 获取详情信息
                    **/
                    init(data,type) {
                        var vm = this;
                        vm.deviceType = data;
                    },
                    /*确定导入*/
                    addEdit() {
                        var vm = this,
                            params={
                                token:omctoken,
                                userId:vm.sasImportForm.userid,
                                excelFile:vm.sasImportForm.excelFile,
                                deviceType:vm.deviceType
                            };
                        params.cpiId = sessionStorage.getItem('sasCpiId') ? sessionStorage.getItem('sasCpiId') : '';
                        params.cpiName = sessionStorage.getItem('sasCpiName') ? sessionStorage.getItem('sasCpiName') : '';
                        params.keyData = sessionStorage.getItem('keyData') ? sessionStorage.getItem('keyData') : '';
                        vm.$refs.sasImportForm.validate((valid) => {
                            if (valid){
                                if (!vm.fileName) {
                                    vm.selectFlag = true
                                    return
                                }
                                vm.uploadFiles(vm.sasImportForm.uploadFileURL, params)
                            } else {
                               return false;
                            }
                        })
                    },
                    /*
                    * 导入函数
                    * url：当前的修改或者添加url
                    * params： 所有的from参数
                    */
                    uploadFiles(url, params, cb) {
                        var vm = this,
                            xhr = new XMLHttpRequest(),
                            fmd = new FormData();
                        if (params) {
                            for (var key in params) {
                                fmd.append(key, params[key]);
                            }
                        }
                        xhr.onreadystatechange = function () {
                            if (this.readyState == 4 && xhr.status == 200) {
                                var data = JSON.parse(xhr.responseText);
                                if (cb && typeof cb == 'function') cb(data);
                            }
                        }
                        xhr.open('post', url);
                        xhr.send(fmd);
                        xhr.onreadystatechange = function () {
                            if (xhr.response) {
                                let str = xhr.response;
                                str = JSON.parse(str)
                                if (str.success) {
                                    vm.$message.success('<%=rb.getString("ChengGong")%>')
                                    vm.close()
                                } else {
                                    vm.$message.error(str.message)
                                }
                            }
                        }
                    },
                    close() { // 关闭 带提示
                      	eventBus.$emit('close-dialogs');
                    },
                    /**
                     *选择文件后，校验格式，并赋值页面显示 
                     *@param file：文件名称
                     */
                    fileChange(file, fileList) {
                        var vm = this;
                        vm.selectFlag = false;
                        var typeFlag = file.name.substr(file.name.lastIndexOf(".")) === '.xls' || file.name.substr(file.name.lastIndexOf(".")) === '.xlsx'
                        vm.typeFlag = typeFlag;
                        if (typeFlag) {
                            vm.sasImportForm.excelFile = file.raw;
                            vm.fileName = file.name;
                        } else {
                            vm.fileName = '';
                        }
                    },
                   
                    checkFile(res, file) {    //发送请求，校验device文件内容 
                        var vm = this;
                        if (res.success) {
                            if (res.suc_count > 0) {
                                vm.$message({
                                    type: 'success',
                                    message: '<%=rb.getString("ChengGong")%>'
                                });
                            } else {
                                vm.$message({
                                    type: 'warning',
                                    message: '<%=rb.getString("ShiBai")%>'
                                });
                            }
                        } else {
                            vm.$message({
                                type: 'error',
                                message: res.msg
                            });
                        }
                        //修改已选择文件状态  
                        var fileList = vm.$refs.uploadCert.uploadFiles;
                        fileList.forEach(function (file) {
                            file.status = 'ready';
                        })
                    },

                    closeFileSelect() { // 关闭文件选择 暂时没有用到 后期维护代码 去掉uploadFiles会用到
                        var vm = this;
                        vm.fileName = '';
                        vm.typeFlag = true;
                        vm.selectFlag = false;
                        vm.$refs.uploadCert.clearFiles();
                    },
                    fileSelect() {  /// 导入文件按钮
                        var vm = this;
                        vm.$refs.uploadCert.clearFiles();
                        vm.$refs['file_up'].click();
                    },
                    exportTemp(){
                         exportByForm("${ctx}/cell/SAS/downloadImportInstallParamTemplate.action",{deviceType:this.deviceType})
                         
                    },
                },

                mounted() {
                    eventBus.$off('hander-rows').$on('hander-rows',this.init);
                }
            })
        </script>