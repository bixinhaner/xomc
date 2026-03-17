<%@ page contentType="text/html;charset=UTF-8" %>
    <%@ include file="/common/taglibs.jsp" %>
        <style>
            #viewPciTaskEnb .group-title {
                margin-bottom: 20px
            }
            
            #viewPciTaskEnb .slide-content {
                padding: 0px !important
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
                width: 130px;
                float: left;
                line-height: 25px
            }
            
            #viewPciTaskEnb .ml20 {
                margin-left: 20px
            }
            
            #viewPciTaskEnb .w290 {
                width: 200px
            }
            
            #viewPciTaskEnb .w350 {
                width: 350px
            }
            
            #viewPciTaskEnb .mt20 {
                margin-top: 20px
            }
            
            #viewPciTaskEnb .w150 {
                width: 150px
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
                border-top: #EEEEEE solid 1px;
                border-right: none;
                border-left: none;
                border-bottom: none;
                position: absolute;
                bottom: 0px;
                left: 0px;
                height: 48px;
                line-height: 48px;
            }
            
            .footer .lnkbuttonGroup {
                margin-left: 48px
            }
            
            .el-form-item__error,
            .el-upload__tip {
                margin-left: 130px
            }
            
            .el-radio-group {
                line-height: 28px;
                margin-top: 3px
            }
            
            .el-icon-circle-add:before {
                font-size: 20px
            }
            
            #viewPciTaskEnb .el-form-item {
                margin-bottom: 22px
            }
            .promptOne{
                font-size: 12px;
                color: #BBBBBB;
                display: inline-block;
                position: relative;
                top: 0px;
            }
            .promptOne .el-icon:before{
                color: #BBBBBB;
                font-size: 12px;
            }
        </style>
        <div id='viewPciTaskEnb'>
            <el-form label-position="top" ref="settingsForm" :model='settingsForm' :rules='rules'>
                <div class="plr15">
                    <el-form-item label="<%=rb.getString("SASProvider") %>" prop="provider">
                        <el-input v-model="settingsForm.provider" placeholder='<%=rb.getString("QingShuRu")%><%=rb.getString("SASProvider")%>' class="w290"
                            maxlength='50' :disabled='editDisabled'></el-input>
                    </el-form-item>
                    <el-form-item label="<%=rb.getString("SASServerURL") %>" prop="serverUrl">
                        <el-input v-model="settingsForm.serverUrl" placeholder='<%=rb.getString("QingShuRu")%><%=rb.getString("SASServerURL")%>'
                            class="w350" maxlength='200'></el-input>
                    </el-form-item>
                </div>
                <div class="form-group last">
                    <div class="group-title">
                        <span class="title-icon"></span><span class="title-text"><%=rb.getString("TLSCertificate")%></span>
                    </div>
                    <div class="plr15">
                        <el-form-item label="<%=rb.getString("CertificateFile")%>" prop="status">
                            <el-radio-group v-model='settingsForm.type' @change='changeType'>
                                <el-radio label='pem'>.PEM</el-radio>
                                <el-radio label='p12'>.P12</el-radio>
                            </el-radio-group>
                        </el-form-item>
                        <el-form-item label="<%=rb.getString("Cert")%>">
                            <el-upload :on-success='checkFile' :on-change="fileChange" :show-file-list=false ref="uploadCert" :action="settingsForm.uploadFileURL"
                                :auto-upload="false">
                                <el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' class="w350">
                                    <a slot="suffix" class="el-icon el-icon-operation-import importBox-fileSlect" @click="fileSelect"></a>
                                </el-input>
                                <span class="clor9" v-if="settingsForm.type === 'pem'"><%=rb.getString("PemOrCrtFormat")%></span>
                                <span class="clor9" v-else><%=rb.getString("P12Format")%></span>
                                <div slot="tip" class="el-form-item__error" v-show="!typeFlag">
                                    {{errorTip}}
                                </div>
                                <div slot="tip" class="el-form-item__error" v-show="selectFlag">
                                    <%=rb.getString("QingXianXuanZeWenJian")%>
                                </div>
                                <a slot="trigger" ref="file_up"></a>
                            </el-upload>
                        </el-form-item>
                        <el-form-item label="<%=rb.getString("PrivateKey")%>" v-show='showBox'>
                            <el-upload :on-success='checkFile2' :on-change="fileChangePrivateKey" :show-file-list=false ref="uploadCert" :action="settingsForm.uploadFileURL"
                                :data="fileParams" :auto-upload="false">
                                <el-input :readonly="true" :value=prvikeName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' class="w350">
                                    <a slot="suffix" class="el-icon el-icon-operation-import importBox-fileSlect" @click="fileSelectPrivateKey"></a>
                                </el-input>
                                <span class="clor9"> <%=rb.getString("PemFormat")%></span>
                                <div slot="tip" class="el-form-item__error" v-show="!privateKey.typeFlag">
                                    <%=rb.getString("PemFormat")%>
                                </div>
                                <div slot="tip" class="el-form-item__error" v-show="privateKey.selectFlag">
                                    <%=rb.getString("QingXianXuanZeWenJian")%>
                                </div>
                                <a slot="trigger" ref="file_up2"></a>
                            </el-upload>
                        </el-form-item>
                        <el-form-item label="<%=rb.getString("MiMa")%>" prop='password'>
                            <el-password v-model="settingsForm.password" size="mini" placeholder="" class="w150" maxlength='50' show-password></el-password>
                            <el-input v-model="settingsForm.password" style="display:none;"></el-input>
                            <div class="promptOne"><span class="el-icon-circle-info el-icon" style="margin-right:10px;" ></span><span><%=rb.getString("ZhengShuCunZaiMiMaTiShi")%></span></div>
                        </el-form-item>
                    </div>
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
                    let providerValidator = (rule, value, cb) => {
                        if (value === '') {
                            cb('<%=rb.getString("SASProvider") %> <%=rb.getString("BuNengWeiKong") %>');
                        } else {
                            cb()
                        }
                    };
                    let serverUrlValidator = (rule, value, cb) => {
                        // var reg = /^((ht|f)tps?):\/\/[\w\-]+(\.[\w\-]+)+([\w\-\.,@?^=%&:\/~\+#]*[\w\-\@?^=%&\/~\+#])?$/
                        if (value === '') {
                            cb('<%=rb.getString("SASServerURL") %> <%=rb.getString("BuNengWeiKong") %>');
                        } else {
                            cb()
                        }
                    };
                    let passwordValidator = (rule, value, cb) => {
                        if (value === '') {
                            //  if(this.settingsForm.type == 'pem'){
                            //     cb('<%=rb.getString("QingShuRuMiMa") %>');
                            // }else{
                            //     cb();
                            // }
                            cb()
                        } else {
                            cb()
                        }
                    };
                    return {
                        settingsForm: {
                            provider: '',
                            serverUrl: '',
                            type: 'pem',
                            password: '',
                            uploadFileURL: '',
                        },
                        privateKey: {
                            typeFlag: true,
                            selectFlag: false
                        },
                        fileParams: {},
                        selectFlag: false,        //标识是否选择了文件 
                        typeFlag: true,
                        fileName: '',
                        prvikeName: '',
                        showBox: true,
                        fileObj: {},
                        infoType: '', // 判断是 新增还是修改
                        rules: {
                            provider: [
                                { validator: providerValidator },
                            ],
                            serverUrl: [
                                { validator: serverUrlValidator }
                            ],
                            password: [
                                { validator: passwordValidator }
                            ]
                        },
                        errorTip: "",
                        editDisabled: false

                    }
                },
                methods: {
                    /**
                    * 获取详情信息
                    **/
                    init(data, type) {
                        var vm = this;
                        vm.infoType = type;

                        if (type === 'add') {
                            vm.settingsForm.uploadFileURL = '${ctx}/cell/SAS/provider/insertProvider.action'
                        } else {
                            vm.settingsForm.uploadFileURL = '${ctx}/cell/SAS/provider/updateProvider.action'
                            vm.settingsForm.provider = data.providerName;
                            vm.settingsForm.serverUrl = data.url;
                            vm.fileParams.id = data.id;
                            vm.editDisabled = true

                        }
                        setTimeout(function () {
                            initForm(vm.$refs.settingsForm);
                        }, 100)
                    },
                    /*确定导入*/
                    addEdit() {
                        var provider;
                        var serverUrl;
                        var vm = this,
                            type = this.settingsForm.type,
                            url = '';
                        password = '';
                        vm.fileParams.token = omctoken;
                        vm.fileParams.providerName = vm.settingsForm.provider;
                        vm.fileParams.url = vm.settingsForm.serverUrl;
                        vm.fileParams.password = vm.settingsForm.password;
                       
                        vm.$refs.settingsForm.validateField('provider', errorMesage => {
                            if (!errorMesage) {
                                provider = true
                            } else {
                                provider = false
                            }
                        })
                        vm.$refs.settingsForm.validateField('serverUrl', errorMesage => {
                            if (!errorMesage) {
                                serverUrl = true
                            } else {
                                serverUrl = false
                            }
                        })
                        if (vm.settingsForm.type === 'p12') { // p12模式下 只验证provider，url 和fileName 有的话就是 true 为空就提示

                            if (vm.infoType === 'add') { // 如果是添加 所有的东西都要校验
                                if (provider && serverUrl && vm.fileName) {
                                    this.uploadFiles(vm.settingsForm.uploadFileURL, vm.fileParams)
                                } else {
                                    if (!vm.fileName) {
                                        vm.selectFlag = true
                                    }
                                }
                            } else { // 如果是修改  可以不用上传文件 其他必须校验
                                if (provider && serverUrl) {
                                    this.uploadFiles(vm.settingsForm.uploadFileURL, vm.fileParams)
                                }
                            }
                            //   if(provider && serverUrl && vm.fileName){
                            //        if (vm.infoType === 'add') {
                            //               this.uploadFiles(vm.settingsForm.uploadFileURL, vm.fileParams)
                            //           } else {
                            //               this.uploadFiles(vm.settingsForm.uploadFileURL, vm.fileParams)
                            //           }
                            //   }else{
                            //     if(!vm.fileName){
                            //         vm.selectFlag = true
                            //       }
                            //   }this.uploadFiles(vm.settingsForm.uploadFileURL, vm.fileParams)
                        } else {
                            if (vm.infoType === 'add') {
                                vm.$refs.settingsForm.validate((valid) => {
                                    if (valid) {
                                        if (!vm.fileName) {
                                            vm.selectFlag = true
                                        }
                                        if (!vm.prvikeName) {
                                            vm.privateKey.selectFlag = true
                                        }
                                        if(vm.fileName && vm.prvikeName){
                                            this.uploadFiles(vm.settingsForm.uploadFileURL, vm.fileParams)
                                        }
                                    } else {
                                        if (!vm.fileName) {
                                            vm.selectFlag = true
                                        }
                                        if (!vm.prvikeName) {
                                            vm.privateKey.selectFlag = true
                                        }
                                    }
                                })
                            } else {
                                if (provider && serverUrl) {
                                    this.uploadFiles(vm.settingsForm.uploadFileURL, vm.fileParams)
                                }
                            }

                        }


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
                                if (cb && typeof cb == 'funceion') cb(data);
                            }
                        }
                        xhr.open('post', url);
                        xhr.send(fmd)
                        xhr.onreadystatechange = function () {
                            if (xhr.response) {
                                let str = JSON.parse(xhr.response)
                                if (str["success"] == true) {
                                    vm.$message.success('<%=rb.getString("ChengGong")%>')
                                    settingsVue.$refs.ctableUpsList.refresh();
                                    settingsVue.$refs.slider.hide();
                                } else {
                                    vm.$message.error(str["message"])
                                }
                            }
                        }
                    },
                    close() { // 关闭 带提示
                        var vm = this;
                        confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
                        if (isFormChanged(vm.$refs.settingsForm)) {
                            vm.$confirm(confirmStr, '<%=rb.getString("QueRen")%>', {
                                confirmButtonText: '<%=rb.getString("QueDing")%>',
                                cancelButtonText: '<%=rb.getString("QuXiao")%>',
                                type: 'warning',
                                closeOnClickModal: false
                            }).then(() => {
                                settingsVue.$refs.slider.hide(); //调取父组件的silder关闭掉
                            }).catch(() => {
                            })
                        } else {
                            settingsVue.$refs.slider.hide();
                        }
                    },
                    /**
                     *选择文件后，校验格式，并赋值页面显示 
                     *@param file：文件名称
                     */
                    fileChange(file, fileList) {

                        var vm = this;
                        vm.selectFlag = false;
                        var typeFlag
                        if (vm.settingsForm.type === 'pem') {
                            typeFlag = file.name.substr(file.name.lastIndexOf(".")) === '.pem' || file.name.substr(file.name.lastIndexOf(".")) === '.crt'
                            vm.errorTip = '<%=rb.getString("PemOrCrtFormat")%>';
                        } else {
                            typeFlag = file.name.substr(file.name.lastIndexOf(".")) === '.p12'
                            vm.errorTip = '<%=rb.getString("P12Format")%>';

                        }

                        vm.typeFlag = typeFlag;
                        if (typeFlag) {
                            vm.fileName = file.name;
                            if (vm.settingsForm.type == 'pem') {
                                vm.fileParams.pemOrCrtCertFile = file.raw;
                            } else {
                                vm.fileParams.p12CertFile = file.raw;
                            }

                        } else {
                            vm.fileName = '';
                        }
                    },
                    /**
                    * PrivateKey 导入监听
                    * file:文件
                    *fileList：文件列表
                    */
                    fileChangePrivateKey(file, fileList) {
                        var vm = this;
                        vm.privateKey.selectFlag = false;
                        const typeFlag = file.name.substr(file.name.lastIndexOf(".")) === '.pem'
                        vm.privateKey.typeFlag = typeFlag;

                        if (typeFlag) {
                            vm.fileParams.privateKeyFile = file.raw;
                            vm.prvikeName = file.name;
                        } else {
                            vm.prvikeName = '';
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

                    checkFile2(res, file) {    //发送请求，校验device文件内容 
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
                    fileSelectPrivateKey() {  //PrivateKey 导入文件按钮
                        var vm = this;
                        vm.$refs.uploadCert.clearFiles();
                        vm.$refs['file_up2'].click();
                    },
                    /**
                    *radio 监听
                    *value :传入值
                    */
                    changeType(value) {
                        var vm = this;
                        type = this.settingsForm.type;
                        vm.$nextTick(() => { //切换的时候清空验证信息
                            vm.$refs.settingsForm.clearValidate()
                        })
                        vm.fileName = '';
                        vm.prvikeName = '';
                        vm.settingsForm.password = '';
                        
                        if (type === 'pem') {
                            vm.typeFlag = true;
                            vm.showBox = true;
                            vm.selectFlag = false;
                            vm.privateKey.selectFlag = false;
                            delete vm.fileParams.p12CertFile
                        } else {
                            vm.typeFlag = true;
                            vm.showBox = false;
                            vm.selectFlag = false;
                            vm.privateKey.selectFlag = false;
                            delete vm.fileParams.pemOrCrtCertFile;
                            delete vm.fileParams.privateKeyFile
                        }
                    }
                },

                mounted() {
                    eventBus.$off('open-dialog').$on('open-dialog', this.init);
                    eventBus.$off('cancel-user').$on('cancel-user', this.close);
                }
            })
        </script>